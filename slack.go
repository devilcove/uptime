package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// slack types
type SlackNotifier struct {
	Name    string
	Token   string
	Channel string
}

type ChannelResponse struct {
	Ok               bool             `json:"ok"`
	Channels         []Channels       `json:"channels"`
	ResponseMetadata ResponseMetadata `json:"response_metadata"`
}

type Topic struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int    `json:"last_set"`
}

type Purpose struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int    `json:"last_set"`
}
type Channels struct {
	ID                      string   `json:"id"`
	Name                    string   `json:"name"`
	IsChannel               bool     `json:"is_channel"`
	IsGroup                 bool     `json:"is_group"`
	IsIm                    bool     `json:"is_im"`
	IsMpim                  bool     `json:"is_mpim"`
	IsPrivate               bool     `json:"is_private"`
	Created                 int      `json:"created"`
	IsArchived              bool     `json:"is_archived"`
	IsGeneral               bool     `json:"is_general"`
	Unlinked                int      `json:"unlinked"`
	NameNormalized          string   `json:"name_normalized"`
	IsShared                bool     `json:"is_shared"`
	IsOrgShared             bool     `json:"is_org_shared"`
	IsPendingExtShared      bool     `json:"is_pending_ext_shared"`
	PendingShared           []any    `json:"pending_shared"`
	ContextTeamID           string   `json:"context_team_id"`
	Updated                 int64    `json:"updated"`
	ParentConversation      any      `json:"parent_conversation"`
	Creator                 string   `json:"creator"`
	IsExtShared             bool     `json:"is_ext_shared"`
	SharedTeamIds           []string `json:"shared_team_ids"`
	PendingConnectedTeamIds []any    `json:"pending_connected_team_ids"`
	IsMember                bool     `json:"is_member"`
	Topic                   Topic    `json:"topic"`
	Purpose                 Purpose  `json:"purpose"`
	PreviousNames           []any    `json:"previous_names"`
	NumMembers              int      `json:"num_members"`
}
type ResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

type SlackMessage struct {
	Channel     string       `json:"channel"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Pretext string `json:"pretext"`
	Text    string `json:"text"`
}

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

func sendSlackStatusNotification(notification []byte, status Status) error {
	var slack SlackNotifier
	if err := json.Unmarshal(notification, &slack); err != nil {
		return err
	}
	data := SlackMessage{
		Text: "Uptime Status Update",
		Attachments: []Attachment{
			{
				Pretext: status.Site,
				Text:    status.URL,
			},
			{
				Pretext: "Status",
				Text:    status.Status,
			},
		},
	}
	return slack.Send(data)
}

func sendSlackTestNotification(notification []byte) error {
	var slack SlackNotifier
	if err := json.Unmarshal(notification, &slack); err != nil {
		return err
	}
	data := SlackMessage{
		Text: "Test Message",
		Attachments: []Attachment{
			{
				Pretext: "Pretext",
				Text:    "first message",
			},
			{
				Pretext: "Pretext2",
				Text:    "second messge",
			},
		},
	}
	return slack.Send(data)
}

func sendSlackCertExpiryNotification(notification []byte, status Status) error {
	var slack SlackNotifier
	if err := json.Unmarshal(notification, &slack); err != nil {
		return err
	}
	data := SlackMessage{
		Text: "Uptime Certificate Expiry",
		Attachments: []Attachment{
			{
				Pretext: status.Site,
				Text:    status.URL,
			},
			{
				Pretext: "Cert Expiry",
				Text:    strconv.Itoa(status.CertExpiry),
			},
		},
	}
	return slack.Send(data)
}
