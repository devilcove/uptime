package templates

import "github.com/a-h/templ"

type ButtonLink struct {
	Name     string
	Location string
}

type StatusRows struct {
	Site         templ.Component
	Status       string
	StatusCode   string
	Time         string
	ResponseTime string
	CertExpiry   string
	Action       []templ.Component
}

type User struct {
	Name    string
	Admin   bool
	Actions []templ.Component
}

type Notification struct {
	Name    string
	Type    string
	Checked bool
	//slack
	Token   string
	Channel string
}

type Monitor struct {
	Name          string
	URL           string
	Freq          string
	Timeout       string
	Notifications []string
	Type          string
}

type History struct {
	Site         string
	Status       string
	StatusCode   string
	Time         string
	ResponseTime string
	CertExpiry   string
}
