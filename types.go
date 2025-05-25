package main

import (
	"time"

	"github.com/gorilla/sessions"
)

type (
	MonitorType string
	NotifyType  string
)

const (
	HTTP MonitorType = "http"
	PING MonitorType = "ping"
	TCP  MonitorType = "tcp"
)

const (
	Slack   NotifyType = "slack"
	Discord NotifyType = "discord"
	Email   NotifyType = "email"
	SMS     NotifyType = "sms"
)

const (
	DiscordRed  = 14177041
	DiscordBlue = 1127128
)

type Status struct {
	Site         string
	URL          string
	Time         time.Time
	StatusCode   int
	Status       string
	CertExpiry   int
	ResponseTime time.Duration
}

//type Monitorer interface {
//	updateStatus()
//	check() Status
//	sendStatusNotification(Status)
//	sendCertificateExpiryNotification(Status)
//}

type Monitor struct {
	Type      MonitorType
	URL       string
	Freq      string
	Name      string
	Timeout   string
	StatusOK  int
	Notifiers []string
}

type TimeFrame int

const (
	Hour TimeFrame = iota
	Day
	Week
	Month
	Year
)

var TimeFrameNames = map[TimeFrame]string{
	Hour:  "Hour",
	Day:   "Day",
	Week:  "Week",
	Month: "Month",
	Year:  "Year",
}

func (t TimeFrame) Name() string {
	return TimeFrameNames[t]
}

type Session struct {
	User     string
	LoggedIn bool
	Admin    bool
	Session  *sessions.Session
}

type User struct {
	Name  string
	Pass  string
	Admin bool
}

// discord types
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

// slack types
type SlackNotifier struct {
	Name    string
	Token   string
	Channel string
}

type Notification struct {
	Name         string
	Type         NotifyType
	Notification any
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
