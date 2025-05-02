package main

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
	URL     string
	Freq    string
	Name    string
	Timeout string
	Status  Status
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

type StatusData struct {
	Title string
	User  string
	Admin bool
	Page  string
	Site  string
	Data  []any
}

type User struct {
	Name  string
	Pass  string
	Admin bool
}
