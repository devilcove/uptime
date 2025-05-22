package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

var errNotFound = errors.New("not found")

func (slack *SlackNotifier) Send(data SlackMessage) error {
	channel, err := slack.getChannel()
	if err != nil {
		log.Println("get slack channel", err)
		return err
	}
	data.Channel = channel
	payload, err := json.Marshal(data)
	if err != nil {
		log.Println("marshal slack Message", err)
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewBuffer(payload))
	if err != nil {
		log.Println("new request", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+slack.Token)
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	resp.Body.Close()
	return err
}

func (s *SlackNotifier) getChannel() (string, error) {
	var channel string
	req, err := http.NewRequest(http.MethodGet, "https://slack.com/api/conversations.list", nil)
	if err != nil {
		log.Println("request", err)
		return channel, err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("http error", err)
		return channel, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("read body", err)
		return channel, err
	}
	var response ChannelResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Println("unmarshal body", err)
		return channel, err
	}
	for _, c := range response.Channels {
		if c.Name == s.Channel {
			return c.ID, nil
		}
	}
	log.Println(string(body))
	return channel, errNotFound
}
