package main

import (
	"encoding/json"
	"log"
	"net/http"
	"syscall"
)

func notifications(w http.ResponseWriter, r *http.Request) {
	data := StatusData{
		Title: "Notifications",
		Page:  "notifications",
	}
	for _, notification := range getAllNotifications() {
		data.Data = append(data.Data, notification)
	}
	log.Println(data)
	showTemplate(w, data)
}
func notify(w http.ResponseWriter, r *http.Request) {
	data := StatusData{
		Title: "New Notifier",
		Page:  "newNotification",
	}
	showTemplate(w, data)
}

func createNewNotify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch r.FormValue("type") {
	case "slack":
		slack := SlackNotifier{
			Name:    r.FormValue("name"),
			Token:   r.FormValue("token"),
			Channel: r.FormValue("channel"),
		}
		log.Println("create slack notification", slack)
		if err := createNotify(Slack, slack); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		w.Write([]byte("not yet implemented"))
		return
	}
	reset <- syscall.SIGHUP
	http.Redirect(w, r, "/", http.StatusFound)
}

func displayDeleteNotify(w http.ResponseWriter, r *http.Request) {
	notifier := r.PathValue("notify")
	log.Println("delete notification", notifier)
	data := StatusData{
		Title: "Delete Notifier",
		Page:  "deleteNotification",
		Site:  notifier,
	}
	data.Data = append(data.Data, notifier)
	showTemplate(w, data)
}

func deleteNotify(w http.ResponseWriter, r *http.Request) {
	notify := r.PathValue("notify")
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("delete notifier", notify)
	if err := removeNotify(notify); err != nil {
		log.Println("delete notify", notify, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func displayEditNotify(w http.ResponseWriter, r *http.Request) {
	notify := r.PathValue("notify")
	notifyType, notification, err := getNotify(notify)
	if err != nil {
		log.Println("get notifier", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data := StatusData{
		Title: "Edit Notifier",
	}
	switch notifyType {
	case Slack:
		var n SlackNotifier
		if err := json.Unmarshal(notification, &n); err != nil {
			log.Println("get notifier", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data.Page = "editSlackNotification"
		data.Data = append(data.Data, n)
	}
	showTemplate(w, data)
}

func editNotify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("parse form", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	notificationType := r.FormValue("type")
	switch notificationType {
	case "slack":
		editSlackNotification(w, r)
	default:
		http.Error(w, errNotImplemented.Error(), http.StatusBadRequest)
		return
	}
}

func editSlackNotification(w http.ResponseWriter, r *http.Request) {
	notification := SlackNotifier{
		Name:    r.PathValue("notify"),
		Token:   r.FormValue("token"),
		Channel: r.FormValue("channel"),
	}
	if err := updateNotify(Slack, notification); err != nil {
		log.Println("update notificaton", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reset <- syscall.SIGHUP
	http.Redirect(w, r, "/", http.StatusFound)
}
