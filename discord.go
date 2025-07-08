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

// DisordNotifier represents a nofifier for discord.
type DisordNotifier struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// DiscordMessage represents the payload structure used to send messages to a Discord webhook.
type DiscordMessage struct {
	Content  string         `json:"content,omitempty"`
	Username string         `json:"username,omitempty"`
	Embeds   []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents embedded fields in DiscordMessage.
type DiscordEmbed struct {
	Title       string `json:"title,omitempty"`
	Color       int    `json:"color,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

// Send dispatches a message to a discord webhook.
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

func sendDiscordStatusNotification(ctx context.Context, notification []byte, status Status, ok int) error {
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
				Color:       discordBlue,
				URL:         status.URL,
				Description: status.URL,
			},
			{
				Title:       "Status",
				Description: status.Status,
			},
		},
	}
	if status.StatusCode != ok {
		data.Embeds[0].Color = discordRed
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
				Color:       discordRed,
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
				Color:       discordRed,
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
