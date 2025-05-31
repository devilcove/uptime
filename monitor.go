package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

func startMonitors(ctx context.Context, wg *sync.WaitGroup) {
	monitorers, err := getMonitors()
	if err != nil {
		log.Println("get monitors", err)
	} else {
		for _, m := range monitorers {
			log.Println("starting monitor", m.Name)
			wg.Add(1)
			go monitor(ctx, wg, &m)
		}
	}
}

func monitor(ctx context.Context, wg *sync.WaitGroup, m *Monitor) {
	defer wg.Done()
	frequency, err := time.ParseDuration(m.Freq)
	if err != nil {
		log.Printf("invalid frequency for monitor %s, %s, %v", m.Name, m.Freq, err)
		return
	}
	timer := time.NewTimer(time.Second)
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	log.Println("starting monitor", m.Name)
	for {
		select {
		case <-ctx.Done():
			log.Println(m.Name, "shutting down")
			return
		case <-ticker.C:
			m.updateStatus()
		case <-timer.C:
			m.updateStatus()
		}
	}
}

func (m *Monitor) updateStatus() {
	var same bool
	newStatus := m.Check()
	oldStatus, err := getStatus(m.Name)
	if err != nil {
		log.Println("get old Status", m.Name, err)
	}
	if newStatus.Status == oldStatus.Status {
		if newStatus.Time.Sub(oldStatus.Time) < time.Hour {
			same = true //nolint:ineffassign,wastedassign
			log.Println("no change in last hour ... skipping", m.Name)
			return
		}
	} else {
		log.Println("status change", m.Name, "monitor status", oldStatus.Status, "checked status", newStatus.Status)
		m.sendStatusNotification(newStatus)
	}
	if newStatus.CertExpiry < 10 && same {
		m.sendCertExpiryNotification(newStatus)
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

func (m *Monitor) checkHTTP() Status {
	s := Status{
		Site: m.Name,
		URL:  m.URL,
		Time: time.Now(),
	}
	if m.Type != HTTP {
		s.Status = "wrong type for http check" + string(m.Type)
		return s
	}
	timeout, err := time.ParseDuration(m.Timeout)
	if err != nil {
		log.Println("Defaulting to 60 second timeout; configured was", m.Timeout)
		timeout = time.Second * 60
	}
	client := http.Client{Timeout: timeout}
	resp, err := client.Get(m.URL)
	s.ResponseTime = time.Since(s.Time)
	if err != nil {
		s.Status = err.Error()
		return s
	}
	defer resp.Body.Close()
	s.Status = resp.Status
	s.StatusCode = resp.StatusCode
	if len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		s.CertExpiry = int(time.Until(cert.NotAfter).Hours() / 24)
	}
	return s
}

func (m *Monitor) Check() Status {
	switch m.Type {
	case HTTP:
		return m.checkHTTP()
	default:
		log.Println("unimplemented monitor check", m.Name, string(m.Type))
		return Status{}
	}
}

func (m *Monitor) sendStatusNotification(status Status) {
	for _, n := range m.Notifiers {
		kind, notification, err := getNotify(n)
		if err != nil {
			log.Println("get notification for monitor", m.Name, n, err)
			return
		}
		switch kind {
		case Slack:
			err = sendSlackStatusNotification(notification, status)
		case Discord:
			err = sendDiscordStatusNotification(notification, status)
		case MailGun:
			err = sendMailGunStatusNotification(notification, status)
		default:
			err = errInvalidNoficationType
		}
		if err != nil {
			log.Println("send status notification", err)
		}
		log.Println("sent", kind, "status nofication for", status.Site, status.URL, status.Status)
	}
}

func (m *Monitor) sendCertExpiryNotification(status Status) {
	for _, n := range m.Notifiers {
		kind, notification, err := getNotify(n)
		if err != nil {
			log.Println("get notification for monitor", m.Name, n, err)
			return
		}
		switch kind {
		case Slack:
			err = sendSlackCertExpiryNotification(notification, status)
		case Discord:
			err = sendDiscordCertExpiryNotification(notification, status)
		case MailGun:
			err = sendMailGunCertExpiryNotification(notification, status)
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
