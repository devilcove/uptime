package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"syscall"
)

func sessionData(w http.ResponseWriter, r *http.Request) (StatusData, error) {
	session, err := store.Get(r, "devilcove-uptime")
	if err != nil {
		log.Println("session err", err)
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return StatusData{}, err
	}
	return StatusData{
		Admin: session.Values["admin"].(bool),
		User:  session.Values["user"].(string),
	}, nil
}

func addUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user.Name = r.FormValue("name")
	user.Pass = r.FormValue("pass")
	admin := r.FormValue("admin")
	if admin == "on" {
		user.Admin = true
	}
	log.Println("add user", user, admin)
	if err := insertUser(user); err != nil {
		log.Println("add user", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	user.Name = r.PathValue("user")
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user.Pass = r.FormValue("pass")
	admin := r.FormValue("admin")
	if admin == "on" {
		user.Admin = true
	}
	if err := modifyUser(user); err != nil {
		log.Println("add user", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	user := r.PathValue("user")
	if err := removeUser(user); err != nil {
		log.Println("delete user", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user := User{
		Name: r.FormValue("name"),
		Pass: r.FormValue("pass"),
	}
	if !validateUser(user) {
		log.Println("unauthorized user")
		http.Error(w, "unauthozied", http.StatusUnauthorized)
		return
	}
	store.MaxAge(300)
	session, err := store.Get(r, "devilcove-uptime")
	if err != nil {
		log.Println("session err", err)
	}
	session.Values["logged in"] = true
	session.Values["user"] = user.Name
	session.Values["admin"] = checkAdmin(user.Name)
	if err := session.Save(r, w); err != nil {
		log.Println("session save", err)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	monitor := Monitor{
		Name:    r.FormValue("name"),
		URL:     r.FormValue("url"),
		Freq:    r.FormValue("freq"),
		Timeout: r.FormValue("timeout"),
	}
	for key, value := range r.Form {
		if key == "notifications" {
			monitor.Notifiers = append(monitor.Notifiers, value...)
		}
	}
	log.Println(monitor)
	type_, err := strconv.Atoi(r.FormValue("type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	monitor.Type = MonitorType(type_)
	if monitor.Type == PING {
		w.Write([]byte("not implemented yet")) //nolint:errcheck
		return
	}
	if !validateURL(monitor.URL) {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	if err := saveMonitor(monitor, false); err != nil {
		log.Println("new monitor", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reset <- syscall.SIGHUP
	http.Redirect(w, r, "/", http.StatusFound)
}

func editMonitor(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	monitor := Monitor{
		Name:    r.FormValue("name"),
		URL:     r.FormValue("url"),
		Freq:    r.FormValue("freq"),
		Timeout: r.FormValue("timeout"),
	}
	type_, err := strconv.Atoi(r.FormValue("type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for key, value := range r.Form {
		if key == "notifications" {
			monitor.Notifiers = append(monitor.Notifiers, value...)
		}
	}
	monitor.Type = MonitorType(type_)
	if monitor.Type == PING {
		w.Write([]byte("not implemented yet")) //nolint:errcheck
		return
	}
	if !validateURL(monitor.URL) {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	if err := saveMonitor(monitor, true); err != nil {
		log.Println("new monitor", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reset <- syscall.SIGHUP
	http.Redirect(w, r, "/", http.StatusFound)
}

func deleteMonitor(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("delete site", site, r.FormValue("history"))
	if err := removeMonitor(site); err != nil {
		log.Println("delete site", site, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.FormValue("history") != "" {
		if err := deleteHistory(site); err != nil {
			log.Println("delete history", site, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func validateURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if _, err := net.LookupIP(u.Host); err != nil {
		return false
	}
	log.Println(err, u.Scheme, u.Host)
	return true
}

func testNotification(w http.ResponseWriter, r *http.Request) {
	n := r.PathValue("notify")
	kind, notification, err := getNotify(n)
	if err != nil {
		log.Println("could not retrieve notification", n, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch kind {
	case Slack:
		var slack SlackNotifier
		if err := json.Unmarshal(notification, &slack); err != nil {
			log.Println("unmarshal slack notification", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
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
		if err := slack.Send(data); err != nil {
			log.Println("send slack message", err)
			http.Error(w, "error sending slack message"+err.Error(), http.StatusBadRequest)
			return
		}
		w.Write([]byte("slack message sent successfully"))
	}
}
