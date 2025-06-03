package main

import (
	"errors"
	"time"

	"github.com/gorilla/sessions"
)

const (
	cookieName  = "devilcove-uptime"
	httpAddr    = ":8090"
	DiscordRed  = 14177041
	DiscordBlue = 1127128
	centerStyle = "text-align:center; margin-left:auto; margin-right:auto;"
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
	MailGun NotifyType = "mailgun(email)"
	Email   NotifyType = "email"
	SMS     NotifyType = "sms"
)

type TimeFrame string

const (
	Hour  TimeFrame = "Hour"
	Day   TimeFrame = "Day"
	Week  TimeFrame = "Week"
	Month TimeFrame = "Month"
	Year  TimeFrame = "Year"
)

type User struct {
	Name  string
	Pass  string
	Admin bool
}

type Status struct {
	Site         string
	URL          string
	Time         time.Time
	StatusCode   int
	Status       string
	CertExpiry   int
	ResponseTime time.Duration
}

type Monitor struct {
	Type      MonitorType
	URL       string
	Freq      string
	Name      string
	Timeout   string
	StatusOK  int
	Notifiers []string
}

type Notification struct {
	Name         string
	Type         NotifyType
	Notification any
}

type Session struct {
	User     string
	LoggedIn bool
	Admin    bool
	Session  *sessions.Session
}

type Radio struct {
	Value   string
	Label   string
	Checked bool
}

var (
	errInvalidNoficationType = errors.New("invalid notification type")
	errNotFound              = errors.New("not found")
)

type MonitorDisplay struct {
	Name          string
	DisplayStatus bool
	PerCent       float64
	AvgResponse   int64
	History       []History
	Status        Status
}

type History struct {
	Status bool
	Time   time.Time
}
