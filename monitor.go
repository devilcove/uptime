package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

func startMonitors(ctx context.Context, wg *sync.WaitGroup) {
	monitorers, err := getMonitors()
	if err != nil {
		log.Println("get monitors", err)
	} else {
		log.Println("starting monitors")
		for _, m := range monitorers {
			if !m.Active {
				continue
			}
			wg.Add(1)
			go monitor(ctx, wg, &m)
		}
	}
}

func monitor(ctx context.Context, wg *sync.WaitGroup, monitor *Monitor) {
	defer wg.Done()
	frequency, err := time.ParseDuration(monitor.Freq)
	if err != nil {
		log.Printf("invalid frequency for monitor %s, %s, %v", monitor.Name, monitor.Freq, err)
		return
	}
	timer := time.NewTimer(time.Second)
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	log.Println("starting monitor", monitor.Name)
	for {
		select {
		case <-ctx.Done():
			log.Println(monitor.Name, "shutting down")
			return
		case <-ticker.C:
			monitor.updateStatus(ctx)
		case <-timer.C:
			monitor.updateStatus(ctx)
		}
	}
}

func (m *Monitor) updateStatus(ctx context.Context) {
	var same bool
	newStatus := m.Check(ctx)
	oldStatus, err := getStatus(m.Name)
	if err != nil {
		log.Println("get old Status", m.Name, err)
	}
	if newStatus.Status == oldStatus.Status {
		same = true
		if newStatus.Time.Sub(oldStatus.Time) < time.Hour {
			log.Println("no change in last hour ... skipping", m.Name)
			return
		}
	} else {
		log.Println("status change", m.Name, "monitor status", oldStatus.Status, "checked status", newStatus.Status)
		m.sendStatusNotification(ctx, newStatus)
	}
	if newStatus.CertExpiry < 10 && same {
		m.sendCertExpiryNotification(ctx, newStatus)
	}
	bytes, err := json.Marshal(&newStatus)
	if err != nil {
		log.Println("json err", err)
		return
	}
	if err = addKey(m.Name, []string{"status"}, bytes); err != nil {
		log.Println("update database", m.Name, err)
		return
	}
	if err = addKey(newStatus.Time.Format(time.RFC3339),
		[]string{"history", m.Name}, bytes); err != nil {
		log.Println("update history", m.Name, err)
	}
	log.Println("status updated", m.Name, newStatus.Status)
}

func (m *Monitor) checkHTTP(ctx context.Context) Status {
	status := Status{
		Site: m.Name,
		URL:  m.URL,
		Time: time.Now(),
	}
	if m.Type != HTTP {
		status.Status = "wrong type for http check" + string(m.Type)
		return status
	}
	timeout, err := time.ParseDuration(m.Timeout)
	if err != nil {
		log.Println("Defaulting to 60 second timeout; configured was", m.Timeout)
		timeout = time.Second * 60
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		status.Status = err.Error()
		return status
	}
	client := http.Client{Timeout: timeout}
	// client := http.Client{Timeout: timeout}
	var resp *http.Response
	// check a couple of times, eliminate transitory errors.
	for range 3 {
		resp, err = client.Do(req)
		status.ResponseTime = time.Since(status.Time)
		if err == nil {
			break
		}
		log.Println("transitory fail for", req.URL, err)
		time.Sleep(time.Second)
	}
	if err != nil {
		var urlError *url.Error
		status.Status = err.Error()
		// if urlError, ok := err.(*url.Error); ok {
		if errors.As(err, &urlError) {
			status.Status = urlError.Unwrap().Error() + " " + timeout.String()
		}
		return status
	}
	defer resp.Body.Close()
	status.Status = resp.Status
	status.StatusCode = resp.StatusCode
	if len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		status.CertExpiry = int(time.Until(cert.NotAfter).Hours() / 24)
	}
	return status
}

// Check conducts a check for a monitor.
func (m *Monitor) Check(ctx context.Context) Status {
	switch m.Type {
	case HTTP:
		return m.checkHTTP(ctx)
	default:
		log.Println("unimplemented monitor check", m.Name, string(m.Type))
		return Status{}
	}
}

func (m *Monitor) sendStatusNotification(ctx context.Context, status Status) {
	for _, n := range m.Notifiers {
		kind, notification, err := getNotify(n)
		if err != nil {
			log.Println("get notification for monitor", m.Name, n, err)
			return
		}
		switch kind {
		case Slack:
			err = sendSlackStatusNotification(ctx, notification, status)
		case Discord:
			err = sendDiscordStatusNotification(ctx, notification, status, m.StatusOK)
		case MailGun:
			err = sendMailGunStatusNotification(ctx, notification, status)
		default:
			err = errInvalidNoficationType
		}
		if err != nil {
			log.Println("send status notification", err)
		}
		log.Println("sent", kind, "status nofication for", status.Site, status.URL, status.Status)
	}
}

func (m *Monitor) sendCertExpiryNotification(ctx context.Context, status Status) {
	for _, n := range m.Notifiers {
		kind, notification, err := getNotify(n)
		if err != nil {
			log.Println("get notification for monitor", m.Name, n, err)
			return
		}
		switch kind {
		case Slack:
			err = sendSlackCertExpiryNotification(ctx, notification, status)
		case Discord:
			err = sendDiscordCertExpiryNotification(ctx, notification, status)
		case MailGun:
			err = sendMailGunCertExpiryNotification(ctx, notification, status)
		default:
			err = errInvalidNoficationType
		}
		if err != nil {
			log.Println("send cert notification", err)
			return
		}
		log.Println("sent", kind, "certification expired notification for", status.Site)
	}
}
