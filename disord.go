package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// discord types.
type DisordNotifier struct {
	Name string
	URL  string
}

type DiscordMessage struct {
	Content  string         `json:"content,omitempty"`
	Username string         `json:"username,omitempty"`
	Embeds   []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string `json:"title,omitempty"`
	Color       int    `json:"color,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

func (d *DisordNotifier) Send(ctx context.Context, data DiscordMessage) error {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Println("marshal discord message", err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.URL, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("new request", err)
		return err
	}
	log.Println("discord send", d.URL, string(payload))
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	log.Println("status", resp.Status, "response", string(bytes))
	return nil
}

func sendDiscordStatusNotification(ctx context.Context, notification []byte, status Status) error {
	var discord DisordNotifier
	if err := json.Unmarshal(notification, &discord); err != nil {
		return err
	}
	data := DiscordMessage{
		Content:  "Uptime Status Alert",
		Username: "Uptime",
		Embeds: []DiscordEmbed{
			{
				Title:       status.Site,
				Color:       DiscordRed,
				URL:         status.URL,
				Description: status.URL,
			},
			{
				Title:       "Status",
				Description: status.Status,
			},
		},
	}
	return discord.Send(ctx, data)
}

func sendDiscordCertExpiryNotification(ctx context.Context, notification []byte, status Status) error {
	var discord DisordNotifier
	if err := json.Unmarshal(notification, &discord); err != nil {
		return err
	}
	data := DiscordMessage{
		Content:  "Uptime Cert Expiry Alert",
		Username: "Uptime",
		Embeds: []DiscordEmbed{
			{
				Title:       status.Site,
				Color:       DiscordRed,
				URL:         status.URL,
				Description: status.URL,
			},
			{
				Title:       "CertExpiry",
				Description: strconv.Itoa(status.CertExpiry),
			},
		},
	}
	return discord.Send(ctx, data)
}

func sendDiscordTestNotification(ctx context.Context, notification []byte) error {
	var discord DisordNotifier
	if err := json.Unmarshal(notification, &discord); err != nil {
		return err
	}
	data := DiscordMessage{
		Content:  "Test Alert",
		Username: "Uptime",
		Embeds: []DiscordEmbed{
			{
				Title:       "test message",
				Color:       DiscordRed,
				URL:         "https://example.com",
				Description: "https://example.com",
			},
			{
				Title:       "Addition Details",
				Description: "status info",
			},
		},
	}
	return discord.Send(ctx, data)
}
