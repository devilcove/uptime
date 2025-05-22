package main

import (
	"log"
	"net/http"
	"syscall"
)

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
		log.Println("update notification", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reset <- syscall.SIGHUP
	http.Redirect(w, r, "/", http.StatusFound)
}
