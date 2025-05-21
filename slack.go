package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

var errNotFound = errors.New("not found")

// func Send(status Status) error {
// 	slack, err := getSlack()
// 	if err != nil {
// 		return err
// 	}
// 	channel, err := getChannel(slack)
// 	if err != nil {
// 		return err
// 	}
// 	data := SlackMessage{
// 		Channel: channel,
// 		Text:    "website down",
// 		Attachments: []Attachment{
// 			{
// 				Pretext: status.URL,
// 				Text:    status.Status + status.Time.Local().Format(time.RFC822),
// 			},
// 		},
// 	}
// 	if status.StatusCode == 200 {
// 		data.Text = "website up"
// 	}
// 	payload, err := json.Marshal(data)
// 	if err != nil {
// 		return err
// 	}
// 	req, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewBuffer(payload))
// 	if err != nil {
// 		return err
// 	}
// 	req.Header.Set("Authorization", slack.Token)
// 	req.Header.Set("Content-Type", "application/json")
// 	client := http.Client{Timeout: time.Second}
// 	_, err = client.Do(req)
// 	return err
// }

func getChannel(s SlackNotifier) (string, error) {
	var channel string
	req, err := http.NewRequest(http.MethodGet, "https://slack.com/api/converstions.list", nil)
	if err != nil {
		return channel, err
	}
	req.Header.Set("Authorization", s.Token)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return channel, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return channel, err
	}
	var response ChannelResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return channel, err
	}
	for _, c := range response.Channels {
		if c.Name == s.Channel {
			return c.ID, nil
		}
	}
	return channel, errNotFound
}
