package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/devilcove/uptime/templates"
)

func mainPage(w http.ResponseWriter, r *http.Request) {
	//data, err := sessionData(w, r)
	//if err != nil {
	//	return
	//}
	title := "Uptime"

	status, err := getKeys([]string{"status"})
	if err != nil {
		log.Println("get status", err)
		http.Error(w, "unable to access database", http.StatusInternalServerError)
		return
	}
	rows := []templates.StatusRows{}
	for _, stat := range status {
		buf := &bytes.Buffer{}
		templates.LinkButton("Delete", fmt.Sprintf("/edit/%s", stat.Site)).Render(context.Background(), buf)

		row := templates.StatusRows{
			Site:         templates.Link(templ.SafeURL("history/"+stat.Site+"/hour"), stat.Site),
			Status:       stat.Status,
			StatusCode:   strconv.Itoa(stat.StatusCode),
			Time:         stat.Time.Local().Format(time.RFC822),
			ResponseTime: stat.ResponseTime.Round(time.Millisecond).String(),
			CertExpiry:   strconv.Itoa(stat.CertExpiry),
		}
		row.Action = append(row.Action, templates.LinkButton("Edit", fmt.Sprintf("/monitor/edit/%s", stat.Site)))
		row.Action = append(row.Action, templates.LinkButton("Delete", fmt.Sprintf("monitor/delete/%s", stat.Site)))
		rows = append(rows, row)
	}

	buttons := []templates.ButtonLink{
		{
			Name:     "New Monitor",
			Location: "/monitor/new",
		},
		{
			Name:     "Notifications",
			Location: "/notifications",
		},
		{
			Name:     "View Logs",
			Location: "/logs",
		},
		{
			Name:     "Logout",
			Location: "./logout",
		},
		{
			Name:     "User Admin",
			Location: "./admin",
		},
	}
	headings := []string{"Site", "Status", "Code", "Time", "Response Time", "CertExpiry", "Actions"}
	stats := templates.Status(buttons, headings, rows)
	components := []templ.Component{
		stats,
	}
	templates.Layout(title, components).Render(context.Background(), w)
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "devilcove-uptime")
	if err != nil {
		log.Println("session err", err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	store.MaxAge(-1)
	if err := session.Save(r, w); err != nil {
		log.Println("session save", err)
	}
	components := []templ.Component{
		templates.Logout(),
	}
	templates.Layout("Logout", components).Render(context.Background(), w)
}

func displayLogin(w http.ResponseWriter, r *http.Request) {
	components := []templ.Component{
		templates.Login(),
	}
	templates.Layout("Login", components).Render(context.Background(), w)
}

func admin(w http.ResponseWriter, r *http.Request) {
	data, err := sessionData(w, r)
	if err != nil {
		return
	}
	users := []templates.User{}
	if data.Admin {
		logins := getUsers()
		for _, login := range logins {
			user := templates.User{}
			user.Name = login.Name
			user.Actions = append(user.Actions, templates.LinkButton("Edit", fmt.Sprintf("/user/%s", login.Name)))
			user.Actions = append(user.Actions, templates.FormButton("Delete",
				templ.SafeURL(fmt.Sprintf("/user/delete/%s", login.Name))))
			users = append(users, user)
		}

	} else {
		user := templates.User{}
		user.Name = data.User
		user.Actions = append(user.Actions, templates.LinkButton("Edit", fmt.Sprintf("/users/edit/%s", data.User)))
		users = append(users, user)
	}
	components := []templ.Component{
		templates.Admin(data.Admin, users),
	}
	templates.Layout("Admin", components).Render(context.Background(), w)
}

func editUser(w http.ResponseWriter, r *http.Request) {
	data, err := sessionData(w, r)
	if err != nil {
		return
	}
	name := r.PathValue("user")
	user := getUser(name)
	if user.Name == "" {
		log.Println("user not found")
		http.Error(w, "no such user", http.StatusBadRequest)
		return
	}
	usr := templates.User{
		Name:  name,
		Admin: user.Admin,
	}
	components := []templ.Component{
		templates.EditUser(usr, data.Admin),
	}
	templates.Layout("Edit User", components).Render(context.Background(), w)
}

func newMonitor(w http.ResponseWriter, r *http.Request) {
	notifications := []templates.Notification{}
	for _, n := range getAllNotifications() {
		notification := templates.Notification{
			Name: n.Name,
		}
		notifications = append(notifications, notification)
	}
	components := []templ.Component{
		templates.NewMonitor(notifications),
	}
	templates.Layout("Edit User", components).Render(context.Background(), w)
}

func deleteSite(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	log.Println("display delete site", site)
	components := []templ.Component{
		templates.DeleteMonitor(site),
	}
	templates.Layout("Edit User", components).Render(context.Background(), w)
}

func edit(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	monitor, err := getMonitor(site)
	if err != nil {
		log.Println("get monitor", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	allNotifications := getAllNotifications()
	data := templates.Monitor{
		Name:          monitor.Name,
		URL:           monitor.URL,
		Freq:          monitor.Freq,
		Timeout:       monitor.Timeout,
		Type:          monitor.Type.Name(),
		Notifications: monitor.Notifiers,
	}
	notifications := []templates.Notification{}
	for _, n := range allNotifications {
		notifications = append(notifications, templates.Notification{Name: n.Name})
	}
	log.Println(data, notifications)
	components := []templ.Component{
		templates.EditMonitor(data, notifications),
	}
	templates.Layout("Edit Monitor", components).Render(context.Background(), w)
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
	h := []templates.History{}
	for _, hist := range history {
		report := templates.History{
			Site:         hist.Site,
			Status:       hist.Status,
			StatusCode:   strconv.Itoa(hist.StatusCode),
			Time:         hist.Time.Local().Format(time.RFC822),
			ResponseTime: hist.ResponseTime.Round(time.Millisecond).String(),
			CertExpiry:   strconv.Itoa(hist.CertExpiry),
		}
		h = append(h, report)
	}
	components := []templ.Component{
		templates.ShowHistory(site, h),
	}
	templates.Layout("History", components).Render(context.Background(), w)
}

func notifications(w http.ResponseWriter, r *http.Request) {
	notifications := []templates.Notification{}
	for _, n := range getAllNotifications() {
		notifications = append(notifications, templates.Notification{
			Name: n.Name,
			Type: NotifyTypeNames[n.Type],
		})
	}
	components := []templ.Component{
		templates.ShowNotifications(notifications),
	}
	templates.Layout("History", components).Render(context.Background(), w)
}

func displayEditNotify(w http.ResponseWriter, r *http.Request) {
	notify := r.PathValue("notify")
	notifyType, notification, err := getNotify(notify)
	if err != nil {
		log.Println("get notifier", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	components := []templ.Component{}
	switch notifyType {
	case Slack:
		var slack SlackNotifier
		if err := json.Unmarshal(notification, &slack); err != nil {
			log.Println("get notifier", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		components = []templ.Component{
			templates.EditSlackNotification(templates.Notification{
				Name:    slack.Name,
				Token:   slack.Token,
				Channel: slack.Channel,
			}),
		}
	}
	templates.Layout("Notifications", components).Render(context.Background(), w)
}

func displayDeleteNotify(w http.ResponseWriter, r *http.Request) {
	notifier := r.PathValue("notify")
	log.Println("delete notification", notifier)
	components := []templ.Component{
		templates.DeleteNotification(notifier),
	}
	templates.Layout("Delete Notification", components).Render(context.Background(), w)
}

func newNotification(w http.ResponseWriter, r *http.Request) {
	components := []templ.Component{
		templates.NewNotification(),
	}
	templates.Layout("New Notification", components).Render(context.Background(), w)
}

func logs(w http.ResponseWriter, r *http.Request) {
	logs, err := os.ReadFile("uptime.log")
	if err != nil {
		log.Println("get logs", err)
		http.Error(w, "unable to retrieve logs", http.StatusInternalServerError)
		return
	}
	data := []string{}
	lines := strings.Split(string(logs), "\n")
	for i := len(lines) - 1; i > len(lines)-200; i-- {
		if i < 0 {
			break
		}
		data = append(data, lines[i])
	}
	//components := []templ.Component{
	//templates.ShowLogs(data),
	//}
	//templates.Layout("Delete Notification", components).Render(context.Background(), w)
	templates.ShowLogs(data).Render(context.Background(), w)
}
