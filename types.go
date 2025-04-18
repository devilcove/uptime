package uptime

import "time"

type Type int

const (
	HTTP Type = iota
	PING
)

var TypeNames = map[Type]string{
	HTTP: "http",
	PING: "ping",
}

func (t Type) Name() string {
	return TypeNames[t]
}

type Checker func(*Monitor) Status

type Status struct {
	Site       string
	Time       time.Time
	StatusCode int
	Status     string
}

type Monitor struct {
	Type    Type
	Url     string
	Freq    string
	Name    string
	Timeout string
	//Check   func(*Monitor) status
}
