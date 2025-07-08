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

// SlackNotifier represents a slock notification.
type SlackNotifier struct {
	Name    string `json:"name,omitempty"`
	Token   string `json:"token,omitempty"`
	Channel string `json:"channel,omitempty"`
}

// ChannelResponse represents the slack response to a conversation list request.
type ChannelResponse struct {
	Ok               bool             `json:"ok"`
	Channels         []Channels       `json:"channels"`
	ResponseMetadata ResponseMetadata `json:"response_metadata"`
}

// Topic respresents a alack topic.
type Topic struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int    `json:"last_set"`
}

// Purpose represents a slack purpose.
type Purpose struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int    `json:"last_set"`
}

// Channels represents a Slack channels response.
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
	SharedTeamIDs           []string `json:"shared_team_ids"`
	PendingConnectedTeamIDs []any    `json:"pending_connected_team_ids"`
	IsMember                bool     `json:"is_member"`
	Topic                   Topic    `json:"topic"`
	Purpose                 Purpose  `json:"purpose"`
	PreviousNames           []any    `json:"previous_names"`
	NumMembers              int      `json:"num_members"`
}

// ResponseMetadata represents slack metadata response.
type ResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

// SlackMessage respresents a message to be posted to a slack channel.
type SlackMessage struct {
	Channel     string       `json:"channel"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

// Attachment represents a slace attachement.
type Attachment struct {
	Pretext string `json:"pretext"`
	Text    string `json:"text"`
}

// Send transmits a message to a slack channel.
func (slack *SlackNotifier) Send(ctx context.Context, data SlackMessage) error {
	channel, err := slack.getChannel(ctx)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://slack.com/api/chat.postMessage", bytes.NewBuffer(payload))
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

func (slack *SlackNotifier) getChannel(ctx context.Context) (string, error) {
	var channel string
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://slack.com/api/conversations.list", nil)
	if err != nil {
		log.Println("request", err)
		return channel, err
	}
	req.Header.Set("Authorization", "Bearer "+slack.Token)
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
		if c.Name == slack.Channel {
			return c.ID, nil
		}
	}
	log.Println(string(body))
	return channel, errNotFound
}

func sendSlackStatusNotification(ctx context.Context, notification []byte, status Status) error {
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
	return slack.Send(ctx, data)
}

func sendSlackTestNotification(ctx context.Context, notification []byte) error {
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
	return slack.Send(ctx, data)
}

func sendSlackCertExpiryNotification(ctx context.Context, notification []byte, status Status) error {
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
	return slack.Send(ctx, data)
}
