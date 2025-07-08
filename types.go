package main

import (
	"errors"
	"time"

	"github.com/gorilla/sessions"
)

const (
	cookieName  = "devilcove-uptime"
	httpAddr    = ":8090"
	discordRed  = 14177041
	discordBlue = 1127128
	centerStyle = "text-align:center; margin-left:auto; margin-right:auto;"
)

// Generic types.
type (
	// MonitorType represents the kind of monitor.
	MonitorType string
	// NotifyType represents the kind of notifications.
	NotifyType string
)

// Monitor types.
const (
	HTTP MonitorType = "http" // http.
	PING MonitorType = "ping" // ping.
	TCP  MonitorType = "tcp"  // tcp.
)

// Notification types.
const (
	Slack   NotifyType = "slack"          // slack.
	Discord NotifyType = "discord"        // discord.
	MailGun NotifyType = "mailgun(email)" // mailgun.
	Email   NotifyType = "email"          // email.
	SMS     NotifyType = "sms"            // sms.
)

// TimeFrame represents a timeframe, e.g. hour, day, week, month, year.
type TimeFrame string

const (
	hour  TimeFrame = "Hour"
	day   TimeFrame = "Day"
	week  TimeFrame = "Week"
	month TimeFrame = "Month"
	year  TimeFrame = "Year"
	all   TimeFrame = "All"
)

// User represents a user of the system.
type User struct {
	Name  string `json:"name,omitempty"`
	Pass  string `json:"pass,omitempty"`
	Admin bool   `json:"admin,omitempty"`
}

// Status represents the current status of an endpoint monitor.
type Status struct {
	Site         string        `json:"site,omitempty"`
	URL          string        `json:"url,omitempty"`
	Time         time.Time     `json:"time,omitempty"`
	StatusCode   int           `json:"status_code,omitempty"`
	Status       string        `json:"status,omitempty"`
	CertExpiry   int           `json:"cert_expiry,omitempty"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
}

// Monitor represents an endpoint monitor.
type Monitor struct {
	Type      MonitorType `json:"type,omitempty"`
	URL       string      `json:"url,omitempty"`
	Freq      string      `json:"freq,omitempty"`
	Name      string      `json:"name,omitempty"`
	Timeout   string      `json:"timeout,omitempty"`
	StatusOK  int         `json:"status_ok,omitempty"`
	Active    bool        `json:"active,omitempty"`
	Notifiers []string    `json:"notifiers,omitempty"`
}

// Notification represents a notification.
type Notification struct {
	Name         string     `json:"name,omitempty"`
	Type         NotifyType `json:"type,omitempty"`
	Notification any        `json:"notification,omitempty"`
}

// Session represents a user session.
type Session struct {
	User     string
	LoggedIn bool
	Admin    bool
	Session  *sessions.Session
}

// Radio represents a UI radiobutton.
type Radio struct {
	Value   string
	Label   string
	Checked bool
}

var (
	errInvalidNoficationType = errors.New("invalid notification type")
	errNotFound              = errors.New("not found")
)

// MonitorDisplay represents an endpoint monitor.
type MonitorDisplay struct {
	Name          string
	Active        bool
	DisplayStatus bool
	PerCent       float64
	Status        Status
}

// Details represents the details for an endpoint monitor.
type Details struct {
	Status     []Status
	Response24 int
	Response30 int
	Uptime24   float64
	Uptime30   float64
}

func compact(status []Status) []Status {
	if len(status) == 0 {
		return status
	}
	var compact []Status
	compact = append(compact, status[0])
	cursor := 0
	for _, s := range status[1:] {
		if s.StatusCode != compact[cursor].StatusCode {
			compact = append(compact, s)
			cursor++
		}
	}
	return compact
}
