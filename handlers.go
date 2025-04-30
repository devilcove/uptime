package main

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"

	"github.com/gorilla/sessions"
)

func loggout(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "helloworld")
	if err != nil {
		log.Println("session err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	store.MaxAge(-1)
	if err := session.Save(r, w); err != nil {
		log.Println("session save", err)
	}
	if _, err := w.Write([]byte("Goodbye")); err != nil {
		log.Println("write", err)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	store = sessions.NewCookieStore([]byte("secret"))
	store.MaxAge(120) // TODO change for production
	session, err := store.Get(r, "helloworld")
	if err != nil {
		log.Println("session err", err)
	}
	session.Values["logged in"] = true
	if err := session.Save(r, w); err != nil {
		log.Println("session save", err)
	}
	templates.ExecuteTemplate(w, "login", nil)
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	status, err := GetKeys([]string{"status"})
	if err != nil {
		log.Println("get status", err)
		http.Error(w, "unable to access database", http.StatusInternalServerError)
		return
	}
	data := StatusData{
		Title: "Testing",
		Theme: "indigo",
		Page:  "status",
	}
	for _, stat := range status {
		report := Status{
			Site:       stat.Site,
			Status:     stat.Status,
			StatusCode: stat.StatusCode,
			Time:       stat.Time.Local(),
		}
		data.Data = append(data.Data, report)
	}
	buf := &bytes.Buffer{}
	if err := templates.ExecuteTemplate(buf, "main", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	buf.WriteTo(w)
}

func logs(w http.ResponseWriter, r *http.Request) {
	logs, err := os.ReadFile("uptime.log")
	if err != nil {
		log.Println("get logs", err)
		http.Error(w, "unable to retrieve logs", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(logs); err != nil {
		log.Println("display logs", err)
	}
}

func new(w http.ResponseWriter, r *http.Request) {
	data := StatusData{
		Title: "New Monitor",
		Page:  "new",
	}
	buf := &bytes.Buffer{}
	if err := templates.ExecuteTemplate(buf, "main", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	buf.WriteTo(w)
}

func create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	monitor := Monitor{
		Name:    r.FormValue("name"),
		Url:     r.FormValue("url"),
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
		w.Write([]byte("not implemente yet"))
		return
	}
	if !validateURL(monitor.Url) {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	if err := SaveMonitor(monitor, false); err != nil {
		log.Println("new monitor", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func edit(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	monitor, err := GetMonitor(site)
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
	buf := &bytes.Buffer{}
	if err := templates.ExecuteTemplate(buf, "main", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

func editMonitor(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	monitor := Monitor{
		Name:    r.FormValue("name"),
		Url:     r.FormValue("url"),
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
		w.Write([]byte("not implemente yet"))
		return
	}
	if !validateURL(monitor.Url) {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	if err := SaveMonitor(monitor, true); err != nil {
		log.Println("new monitor", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func delete(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	log.Println("delete", site)
	data := StatusData{
		Title: "New Monitor",
		Page:  "delete",
		Site:  site,
	}
	data.Data = append(data.Data, site)
	buf := &bytes.Buffer{}
	if err := templates.ExecuteTemplate(buf, "main", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	buf.WriteTo(w)
}

func deleteMonitor(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("delete site", site, r.FormValue("history"))
	if err := DeleteMonitor(site); err != nil {
		log.Println("delete site", site, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.FormValue("history") != "" {
		if err := DeleteHistory(site); err != nil {
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
	history, err := GetHistory([]string{"history", site}, timeFrame)
	if err != nil {
		log.Println("get status", err)
		http.Error(w, "unable to access database: "+err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(history)
	data := StatusData{
		Title: "History for " + site,
		Theme: "indigo",
		Page:  "history",
		Site:  site,
	}
	for _, hist := range history {
		report := Status{
			Site:       hist.Site,
			Status:     hist.Status,
			StatusCode: hist.StatusCode,
			Time:       hist.Time.Local(),
		}
		data.Data = append(data.Data, report)
	}
	buf := &bytes.Buffer{}
	if err := templates.ExecuteTemplate(buf, "main", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	buf.WriteTo(w)

}

func validateURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}
