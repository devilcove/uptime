package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/devilcove/uptime"
)

func monitor(m *uptime.Monitor) {
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
	for {
		select {
		case <-ticker.C:
			updateStatus(m, checker)
		case <-timer.C:
			updateStatus(m, checker)
		}
	}
}

func updateStatus(m *uptime.Monitor, check uptime.Checker) {
	status := check(m)
	bytes, err := json.Marshal(&status)
	if err != nil {
		log.Println("json err", err)
		return
	}
	log.Println("updating status", m.Name, status.Status)
	db, err := uptime.OpenDB()
	if err != nil {
		log.Println("open database", m.Name, err)
		return
	}
	defer db.Close()
	if err = uptime.AddKey(db, m.Name, []string{"status"}, bytes); err != nil {
		log.Println("update database", m.Name, err)
		return
	}
	if err = uptime.AddKey(db, status.Time.Format(time.RFC3339),
		[]string{"history", m.Name}, bytes); err != nil {
		log.Println("update history", m.Name, err)
	}
	log.Println("status updated", m.Name, status.Status)
}

func checkHTTP(m *uptime.Monitor) uptime.Status {
	s := uptime.Status{
		Site: m.Name,
		Time: time.Now(),
	}
	if m.Type != uptime.HTTP {
		s.Status = "wrong type for http check" + m.Type.Name()
		return s
	}
	timeout, err := time.ParseDuration(m.Timeout)
	if err != nil {
		log.Println("Defaulting to 60 second timeout; configured was", m.Timeout)
		timeout = time.Duration(time.Second * 60)
	}
	client := http.Client{Timeout: timeout}
	resp, err := client.Get(m.Url)
	if err != nil {
		s.Status = err.Error()
		return s
	}
	s.Status = resp.Status
	s.StatusCode = resp.StatusCode
	return s
}

func getMonitors() []uptime.Monitor {
	return []uptime.Monitor{
		{
			Type:    uptime.HTTP,
			Url:     "https://nusak.ca",
			Freq:    "1m",
			Name:    "nusak",
			Timeout: "2s",
			//Check:   checkHTTP,
		},
		{
			Type:    uptime.HTTP,
			Url:     "https://time.nusak.ca",
			Freq:    "1m",
			Name:    "time",
			Timeout: "2s",
			//Check:   checkHTTP,
		},
	}
}

func getChecker(t uptime.Type) uptime.Checker {
	switch t {
	case uptime.HTTP:
		return checkHTTP
	default:
		return nil
	}
}
