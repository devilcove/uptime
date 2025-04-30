package main

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"slices"

	"github.com/devilcove/uptime"
	"github.com/gorilla/sessions"
	"go.etcd.io/bbolt"
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
	//if r.URL.Path != "/" {
	//http.NotFound(w, r)
	//return
	//}
	db := openDB(w)
	if db == nil {
		return
	}
	defer db.Close()
	status, err := uptime.GetKeys(db, []string{"status"})
	if err != nil {
		log.Println("get status", err)
		http.Error(w, "unable to access database", http.StatusInternalServerError)
		return
	}
	data := uptime.StatusData{
		Title: "Testing",
		Theme: "indigo",
		Page:  "status",
	}
	for _, stat := range status {
		report := uptime.Status{
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
	logs, err := os.ReadFile("../uptimed/uptimed.log")
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
	data := uptime.StatusData{
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
	log.Println(r.FormValue("name"))
	log.Println(r.FormValue("url"))
	log.Println(r.FormValue("type"))
	log.Println(r.FormValue("freq"))
	log.Println(r.FormValue("timeout"))
	http.Redirect(w, r, "/", http.StatusFound)
}

func edit(w http.ResponseWriter, r *http.Request) {

}
func delete(w http.ResponseWriter, r *http.Request) {

}

func history(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	duration := r.PathValue("duration")
	var timeFrame uptime.TimeFrame
	switch duration {
	case "year":
		timeFrame = uptime.Year
	case "month":
		timeFrame = uptime.Month
	case "week":
		timeFrame = uptime.Week
	case "day":
		timeFrame = uptime.Day
	default:
		timeFrame = uptime.Hour
	}
	db := openDB(w)
	if db == nil {
		return
	}
	defer db.Close()
	history, err := uptime.GetHistory(db, []string{"history", site}, timeFrame)
	if err != nil {
		log.Println("get status", err)
		http.Error(w, "unable to access database", http.StatusInternalServerError)
		return
	}
	slices.Reverse(history)
	data := uptime.StatusData{
		Title: "History for " + site,
		Theme: "indigo",
		Page:  "history",
		Site:  site,
	}
	for _, hist := range history {
		report := uptime.Status{
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

func openDB(w http.ResponseWriter) *bbolt.DB {
	//config := uptime.GetConfig()
	//if config == nil {
	//log.Println("no configuration ... bailing")
	//http.Error(w, "invaild server configuration", http.StatusInternalServerError)
	//return nil
	//}
	db, err := uptime.OpenDB()
	if err != nil {
		log.Println("database access", err)
		http.Error(w, "unable to access database", http.StatusInternalServerError)
		return nil
	}
	return db
}
