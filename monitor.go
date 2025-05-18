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
	checker := getChecker(m.Type)
	if checker == nil {
		log.Println("no checker for", m.Name, m.Type)
		return
	}
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
			updateStatus(m, checker)
		case <-timer.C:
			updateStatus(m, checker)
		}
	}
}

func updateStatus(m *Monitor, check Checker) {
	status := check(m)
	if status.Status == m.Status.Status {
		if status.Time.Sub(m.Status.Time) < time.Hour {
			log.Println("no change in last hour ... skipping", m.Name)
			return
		}
	}
	m.Status = status
	bytes, err := json.Marshal(&status)
	if err != nil {
		log.Println("json err", err)
		return
	}
	if err = addKey(m.Name, []string{"status"}, bytes); err != nil {
		log.Println("update database", m.Name, err)
		return
	}
	if err = addKey(status.Time.Format(time.RFC3339),
		[]string{"history", m.Name}, bytes); err != nil {
		log.Println("update history", m.Name, err)
	}
	log.Println("status updated", m.Name, status.Status)
}

func checkHTTP(m *Monitor) Status {
	s := Status{
		Site: m.Name,
		URL:  m.URL,
		Time: time.Now(),
	}
	if m.Type != HTTP {
		s.Status = "wrong type for http check" + m.Type.Name()
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

func getChecker(t Type) Checker {
	switch t { //nolint:exhaustive
	case HTTP:
		return checkHTTP
	default:
		return nil
	}
}
