package main

import (
	"bytes"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
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

func admin(w http.ResponseWriter, r *http.Request) {
	data, err := sessionData(w, r)
	if err != nil {
		return
	}
	data.Page = "admin"
	if data.Admin {
		users := getUsers()
		for _, user := range users {
			log.Println(user)
			data.Data = append(data.Data, user)
		}
	}
	showTemplate(w, data)
}

func editUser(w http.ResponseWriter, r *http.Request) {
	data, err := sessionData(w, r)
	if err != nil {
		return
	}
	name := r.PathValue("user")
	data.Title = "Edit Password"
	data.Page = "editUser"
	user := getUser(name)
	if user.Name == "" {
		log.Println("user not found")
		http.Error(w, "no such user", http.StatusBadRequest)
		return
	}
	data.Data = append(data.Data, user)
	showTemplate(w, data)
}

func newUser(w http.ResponseWriter, r *http.Request) {
	data := StatusData{
		Title: "New User",
		Page:  "newUser",
	}
	showTemplate(w, data)
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

func loggout(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "devilcove-uptime")
	if err != nil {
		log.Println("session err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	store.MaxAge(-1)
	if err := session.Save(r, w); err != nil {
		log.Println("session save", err)
	}
	showTemplate(w, StatusData{Page: "logout"})
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

func displayLogin(w http.ResponseWriter, r *http.Request) {
	data := StatusData{
		Title: "Login",
		Page:  "login",
	}
	showTemplate(w, data)
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	data, err := sessionData(w, r)
	if err != nil {
		return
	}
	data.Title = "Uptime"
	data.Page = "status"
	status, err := getKeys([]string{"status"})
	if err != nil {
		log.Println("get status", err)
		http.Error(w, "unable to access database", http.StatusInternalServerError)
		return
	}
	for _, stat := range status {
		report := Status{
			Site:         stat.Site,
			Status:       stat.Status,
			StatusCode:   stat.StatusCode,
			Time:         stat.Time.Local(),
			ResponseTime: stat.ResponseTime.Round(time.Millisecond),
			CertExpiry:   stat.CertExpiry,
		}
		data.Data = append(data.Data, report)
	}
	showTemplate(w, data)
}

func logs(w http.ResponseWriter, r *http.Request) {
	logs, err := os.ReadFile("uptime.log")
	if err != nil {
		log.Println("get logs", err)
		http.Error(w, "unable to retrieve logs", http.StatusInternalServerError)
		return
	}
	data := StatusData{
		Title: "Logs",
		Page:  "logs",
	}
	lines := strings.Split(string(logs), "\n")
	for i := len(lines) - 1; i > len(lines)-200; i-- {
		if i < 0 {
			break
		}
		data.Data = append(data.Data, lines[i])
	}
	showTemplate(w, data)
}

func new(w http.ResponseWriter, r *http.Request) {
	data := StatusData{
		Title: "New Monitor",
		Page:  "new",
	}
	showTemplate(w, data)
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
	type_, err := strconv.Atoi(r.FormValue("type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	monitor.Type = Type(type_)
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

func edit(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	monitor, err := getMonitor(site)
	if err != nil {
		log.Println("get monitor", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	data := StatusData{
		Title: "Edit Monitor",
		Page:  "edit",
		Site:  site,
	}
	data.Data = append(data.Data, monitor)
	showTemplate(w, data)
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
	monitor.Type = Type(type_)
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
	http.Redirect(w, r, "/", http.StatusFound)
}

func deleteSite(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	log.Println("delete", site)
	data := StatusData{
		Title: "Delete Monitor",
		Page:  "delete",
		Site:  site,
	}
	data.Data = append(data.Data, site)
	showTemplate(w, data)
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

func history(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	duration := r.PathValue("duration")
	var timeFrame TimeFrame
	switch duration {
	case "year":
		timeFrame = Year
	case "month":
		timeFrame = Month
	case "week":
		timeFrame = Week
	case "day":
		timeFrame = Day
	default:
		timeFrame = Hour
	}
	history, err := getHistory([]string{"history", site}, timeFrame)
	if err != nil {
		log.Println("get status", err)
		http.Error(w, "unable to access database: "+err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(history)
	data := StatusData{
		Title: "History: " + site,
		Page:  "history",
		Site:  site,
	}
	for _, hist := range history {
		report := Status{
			Site:         hist.Site,
			Status:       hist.Status,
			StatusCode:   hist.StatusCode,
			Time:         hist.Time.Local(),
			ResponseTime: hist.ResponseTime.Round(time.Millisecond),
			CertExpiry:   hist.CertExpiry,
		}
		data.Data = append(data.Data, report)
	}
	showTemplate(w, data)
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

func showTemplate(w http.ResponseWriter, data StatusData) {
	buf := &bytes.Buffer{}
	if err := templates.ExecuteTemplate(buf, "main", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	if _, err := buf.WriteTo(w); err != nil {
		log.Println(err)
	}
}
