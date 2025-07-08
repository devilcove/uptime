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
	Name       string   `json:"name,omitempty"`
	APIKey     string   `json:"api_key,omitempty"`
	Recipients []string `json:"recipients,omitempty"`
	Domain     string   `json:"domain,omitempty"`
}

// MailGunMessage represents an email to be sent to mailgun server.
type MailGunMessage struct {
	To      []string `json:"to,omitempty"`
	Subject string   `json:"subject,omitempty"`
	From    string   `json:"from,omitempty"`
	Text    string   `json:"text,omitempty"`
}

// SendNotification transmits a message to mailgun server.
func (m *MailGunNotifier) SendNotification(ctx context.Context, msg string) error {
	contentType, body, err := m.form(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mailgun.net/v3/"+m.Domain+"/messages", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
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

func sendMailGunTestNotification(ctx context.Context, notification []byte) error {
	var m MailGunNotifier
	if err := json.Unmarshal(notification, &m); err != nil {
		return err
	}
	return m.SendNotification(ctx, "test message from uptime monitor")
}

func sendMailGunStatusNotification(ctx context.Context, notification []byte, status Status) error {
	var mailgun MailGunNotifier
	if err := json.Unmarshal(notification, &mailgun); err != nil {
		return err
	}
	return mailgun.SendNotification(ctx,
		fmt.Sprintf("Uptime Status Message\n%s %s \nStatus %s",
			status.Site, status.URL, status.Status))
}

func sendMailGunCertExpiryNotification(ctx context.Context, notification []byte, status Status) error {
	var mailgun MailGunNotifier
	if err := json.Unmarshal(notification, &mailgun); err != nil {
		return err
	}
	return mailgun.SendNotification(ctx,
		(fmt.Sprintf("Uptime Certificate Expiry Message\n%s %s\n Certificate will expire in %d days",
			status.Site, status.URL, status.CertExpiry)))
}

func (m *MailGunNotifier) form(msg string) (string, io.Reader, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	defer writer.Close()
	if err := writer.WriteField("from", "uptime@"+m.Domain); err != nil {
		return "", nil, err
	}
	if err := writer.WriteField("to", strings.Join(m.Recipients, ",")); err != nil {
		return "", nil, err
	}
	if err := writer.WriteField("subject", "Uptime Status Alert"); err != nil {
		return "", nil, err
	}
	if err := writer.WriteField("text", msg); err != nil {
		return "", nil, err
	}
	return writer.FormDataContentType(), body, nil
}
