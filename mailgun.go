package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mailgun/mailgun-go/v5"
)

// MailGunNotifier holds data to send emails via MailGun
type MailGunNotifier struct {
	Name       string
	APIKey     string
	Recipients []string
	Domain     string
}

func (m *MailGunNotifier) SendNotification(body string) error {
	mg := mailgun.NewMailgun(m.APIKey)
	message := mailgun.NewMessage(m.Domain, "uptime@"+m.Domain, "Uptime Status Alert",
		body, m.Recipients...)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	resp, err := mg.Send(ctx, message)
	if err != nil {
		log.Println("mailgun send", err)
		return err
	}
	log.Println("mailgun response", resp.ID, resp.Message)
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
