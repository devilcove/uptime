package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// MailGunNotifier holds data to send emails via MailGun.
type MailGunNotifier struct {
	Name       string
	APIKey     string
	Recipients []string
	Domain     string
}

type MailGunMessage struct {
	To      []string `json:"to,omitempty"`
	Subject string   `json:"subject,omitempty"`
	From    string   `json:"from,omitempty"`
	Text    string   `json:"text,omitempty"`
}

func (m *MailGunNotifier) SendNotification(msg string) error {
	ct, body, err := m.Form(msg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mailgun.net/v3/"+m.Domain+"/messages", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", ct)
	req.SetBasicAuth("api", m.APIKey)
	client := http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mailgun error %d %s", resp.StatusCode, string(bytes))
	}
	return nil
}

func sendMailGunTestNotification(notification []byte) error {
	var m MailGunNotifier
	if err := json.Unmarshal(notification, &m); err != nil {
		return err
	}
	return m.SendNotification("test message from uptime monitor")
}

func sendMailGunStatusNotification(notification []byte, status Status) error {
	var mailgun MailGunNotifier
	if err := json.Unmarshal(notification, &mailgun); err != nil {
		return err
	}
	return mailgun.SendNotification(
		fmt.Sprintf("Uptime Status Message\n%s %s \nStatus %s",
			status.Site, status.URL, status.Status))
}

func sendMailGunCertExpiryNotification(notification []byte, s Status) error {
	var mailgun MailGunNotifier
	if err := json.Unmarshal(notification, &mailgun); err != nil {
		return err
	}
	return mailgun.SendNotification(
		(fmt.Sprintf("Uptime Certificate Expiry Message\n%s %s\n Certificate will expire in %d days",
			s.Site, s.URL, s.CertExpiry)))
}

func (m *MailGunNotifier) Form(msg string) (string, io.Reader, error) {
	body := new(bytes.Buffer)
	mp := multipart.NewWriter(body)
	defer mp.Close()
	if err := mp.WriteField("from", "uptime@"+m.Domain); err != nil {
		return "", nil, err
	}
	if err := mp.WriteField("to", strings.Join(m.Recipients, ",")); err != nil {
		return "", nil, err
	}
	if err := mp.WriteField("subject", "Uptime Status Alert"); err != nil {
		return "", nil, err
	}
	if err := mp.WriteField("text", msg); err != nil {
		return "", nil, err
	}
	return mp.FormDataContentType(), body, nil
}
