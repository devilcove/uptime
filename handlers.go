package main

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/kr/pretty"
	"maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

func favicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "files/favicon.svg")
}

func styles(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "files/styles.css")
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	status, err := getAllStatus()
	if err != nil {
		displayError(w, err)
		return
	}
	if err := layout("Testing", []gomponents.Node{
		html.H2(gomponents.Text("Uptime Status")),
		html.Br(nil),
		linkButton("/monitor/new", "New Monitor"),
		linkButton("notifications/", "Notifications"),
		linkButton("/logs", "View Logs"),
		linkButton("/logout", "Logout"),
		linkButton("/user/", "User Admin"),
		html.Br(nil),
		statusTable(status),
	}).Render(w); err != nil {
		log.Println("render main page", err)
	}
}

func logs(w http.ResponseWriter, r *http.Request) { //nolint:revive,varnamelen
	logs, err := os.ReadFile("uptime.log")
	if err != nil {
		log.Println("get logs", err)
		http.Error(w, "unable to retrieve logs", http.StatusInternalServerError)
		return
	}
	data := []gomponents.Node{}
	lines := strings.Split(string(logs), "\n")
	for i := len(lines) - 1; i > len(lines)-200; i-- {
		if i < 0 {
			break
		}
		data = append(data, gomponents.Text(lines[i]), html.Br(nil))
	}

	if err := displayLogs(data).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, cookieName)
	if err != nil {
		log.Println("session error", err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	store.MaxAge(-1)
	if err := session.Save(r, w); err != nil {
		log.Println("session save", err)
	}
	if err := layout("Logout", []gomponents.Node{
		html.H2(gomponents.Text("Goodbye")),
		html.Br(nil),
		linkButton("/", "Home"),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func displayLogin(w http.ResponseWriter, r *http.Request) {
	if err := layout("Login", []gomponents.Node{
		html.Form(html.Class("center"),
			html.Action("/login"),
			html.Method("POST"),
			html.Table(
				inputTableRow("Name", "name", "text", "", "40"),
				inputTableRow("Pass", "pass", "password", "", "40"),
			),
			submitButton("Login"),
		),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
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
	log.Println("user", user.Name, "logged in")
	http.Redirect(w, r, "/", http.StatusFound)
}

func admin(w http.ResponseWriter, r *http.Request) {
	var logins []User
	data, err := sessionData(r)
	if err != nil {
		return
	}
	if data.Admin {
		logins = getUsers()
	} else {
		logins = append(logins, getUser(data.User))
	}
	if err := layout("Admin", []gomponents.Node{
		container(true,
			html.H2(gomponents.Text("Admin Page")),
			gomponents.If(data.Admin,
				html.Button(
					gomponents.Attr("type", "button"),
					gomponents.Attr("onclick", "document.getElementById('new').showModal()"),
					gomponents.Text("Create New User"),
				),
			),
			linkButton("/", "Home"),
			html.Br(nil), html.Br(nil),
			userTable(logins),
		),
		newUserDialog(),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func editUser(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("user")
	user := getUser(name)
	if user.Name == "" {
		displayError(w, errors.New("no such user"))
		return
	}

	if err := layout("Edit User", []gomponents.Node{
		html.H2(gomponents.Text("Edit User " + user.Name)),
		html.Form(html.Class("center"),
			html.Action("/user/"+user.Name),
			html.Method("POST"),
			html.Div(
				html.Label(gomponents.Text("Pass")),
				html.Input(
					html.ID("password"),
					html.Type("password"),
					html.Name("pass"),
					gomponents.Attr("size", "40"),
				),
			),
			html.Div(
				html.Label(gomponents.Text("Admin")),
				checkbox("admin", user.Admin),
			),
			html.Div(
				linkButton("/user/", "Cancel"),
				submitButton("Edit"),
			),
		),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	user := r.PathValue("user")
	if err := removeUser(user); err != nil {
		displayError(w, err)
		return
	}
	http.Redirect(w, r, "/user/", http.StatusFound)
}

func addUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		displayError(w, err)
		return
	}
	user := User{
		Name: r.FormValue("name"),
		Pass: r.FormValue("pass"),
	}
	admin := r.FormValue("admin")
	if admin == "on" {
		user.Admin = true
	}
	log.Println("add user", user, admin)
	if err := insertUser(user); err != nil {
		displayError(w, err)
		return
	}
	http.Redirect(w, r, "/user/", http.StatusFound)
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
	http.Redirect(w, r, "/user/", http.StatusFound)
}

func displayError(w http.ResponseWriter, err error) {
	log.Println(err)
	if err := layout("Error", []gomponents.Node{
		html.H1(gomponents.Text("An Error Occurred")),
		html.P(gomponents.Text(err.Error())),
		linkButton("/", "Home"),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func isAdmin(r *http.Request) bool {
	session, err := sessionData(r)
	if err != nil {
		return false
	}
	return session.Admin
}

func newMonitor(w http.ResponseWriter, r *http.Request) {
	notifications := getAllNotifications()
	var notifyCheckboxes []gomponents.Node
	for _, n := range notifications {
		checkbox := html.Input(
			html.Type("checkbox"),
			html.Name(n.Name),
			gomponents.Text(n.Name),
		)
		notifyCheckboxes = append(notifyCheckboxes, checkbox, gomponents.Text(n.Name))
	}
	if err := layout("New Monitor", []gomponents.Node{
		html.H2(gomponents.Text("Create New Monitor")),
		html.Form(
			html.Method("post"),
			html.Action("/monitor/new"),
			html.Table(
				inputTableRow("Name", "name", "text", "", "60"),
				inputTableRow("URL", "url", "text", "", "60"),
				inputTableRow("OK Status", "statusok", "number", "200", "60"),
				radioGroup("Frequency", "freq", []Radio{
					{"1m", "1 Minute", false},
					{"5m", "5 Minutes", false},
					{"30m", "30 Minutes", false},
					{"60m", "60 Minute", false},
				}),
				radioGroup("Timeout", "timeout", []Radio{
					{"1s", "1 Second", false},
					{"2s", "2 Seconds", false},
					{"5s", "5 Seconds", false},
					{"10s", "10 Seconds", false},
				}),
				radioGroup("Type", "type", []Radio{
					{"http", "Website", false},
					{"ping", "Ping", false},
				}),
				html.Tr(
					html.Td(html.Label(gomponents.Text("Notifications"))),
					html.Td(gomponents.Group(notifyCheckboxes)),
				),
			),
			linkButton("/", "Cancel"),
			submitButton("Create"),
		),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func createMonitor(w http.ResponseWriter, r *http.Request) {
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
		Type:    MonitorType(r.FormValue("type")),
	}
	for key, value := range r.Form {
		if key == "notifications" {
			monitor.Notifiers = append(monitor.Notifiers, value...)
		}
	}
	log.Println(monitor)
	ok, err := strconv.Atoi(r.FormValue("statusok"))
	if err != nil {
		log.Println("ascii conversion statusOK", r.FormValue("statusok"), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	monitor.StatusOK = ok
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
	site := r.PathValue("site")
	monitor, err := getMonitor(site)
	if err != nil {
		displayError(w, err)
		return
	}
	notifications := getAllNotifications()
	var notifyCheckboxes []gomponents.Node
	for _, n := range notifications {
		checkbox := html.Input(
			html.Type("checkbox"),
			html.Name(n.Name),
			gomponents.Text(n.Name),
			gomponents.If(slices.Contains(monitor.Notifiers, n.Name), html.Checked()),
		)
		notifyCheckboxes = append(notifyCheckboxes, checkbox, gomponents.Text(n.Name))
	}
	if err := layout("Edit Monitor", []gomponents.Node{
		html.H2(gomponents.Text("Edit Monitor")),
		html.Form(
			html.Method("post"),
			html.Action("/monitor/edit/"+monitor.Name),
			html.Table(
				html.Tr(
					html.Td(html.Label(html.For("name"), gomponents.Text("Name"))),
					html.Td(html.Input(
						html.Name("name"),
						html.Type("text"),
						html.Required(),
						html.Value(monitor.Name),
						gomponents.Attr("size", "60"),
					)),
				),
				html.Tr(
					html.Td(html.Label(html.For("url"), gomponents.Text("URL"))),
					html.Td(html.Input(
						html.Name("url"),
						html.Type("text"),
						html.Required(),
						html.Value(monitor.URL),
						gomponents.Attr("size", "60"),
					)),
				),
				html.Tr(
					html.Td(html.Label(html.For("statusok"), gomponents.Text("OK Status"))),
					html.Td(html.Input(
						html.Name("statusok"),
						html.Type("number"),
						html.Required(),
						html.Value("200"),
						gomponents.Attr("size", "60"),
					)),
				),
				radioGroup("Frequency", "freq", []Radio{
					{"1m", "1 Minute", monitor.Freq == "1m"},
					{"5m", "5 Minutes", monitor.Freq == "5m"},
					{"30m", "30 Minutes", monitor.Freq == "30m"},
					{"60m", "60 Minute", monitor.Freq == "60m"},
				}),
				radioGroup("Timeout", "timeout", []Radio{
					{"1s", "1 Second", monitor.Timeout == "1s"},
					{"2s", "2 Seconds", monitor.Timeout == "2s"},
					{"5s", "5 Seconds", monitor.Timeout == "5s"},
					{"10s", "10 Seconds", monitor.Timeout == "10s"},
				}),
				radioGroup("Type", "type", []Radio{
					{"http", "Website", monitor.Type == "http"},
					{"ping", "Ping", monitor.Type == "ping"},
				}),
				html.Tr(
					html.Td(html.Label(gomponents.Text("Notifications"))),
					html.Td(gomponents.Group(notifyCheckboxes)),
				),
			),
			linkButton("/", "Cancel"),
			submitButton("Update"),
		),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func deleteSite(w http.ResponseWriter, r *http.Request) {
	monitor := r.PathValue("site")
	if err := layout("Delete Monitor", []gomponents.Node{
		html.H2(gomponents.Text("Delete Monitor " + monitor)),
		html.Form(
			html.Action("/monitor/delete/"+monitor),
			html.Method("post"),
			html.Input(
				html.Type("checkbox"),
				html.Value("history"),
				html.Name("history"),
			),
			html.Label(
				html.For("history"),
				gomponents.Text("Also deleted history?"),
			),
			html.Br(), html.Br(),
			linkButton("/", "Cancel"),
			submitButton("Delete"),
		),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
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

func updateMonitor(w http.ResponseWriter, r *http.Request) {
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
		Type:    MonitorType(r.FormValue("type")),
	}
	ok, err := strconv.Atoi(r.FormValue("statusok"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for key, value := range r.Form {
		if key == "notifications" {
			monitor.Notifiers = append(monitor.Notifiers, value...)
		}
	}
	monitor.StatusOK = ok
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
	if err := layout("History", []gomponents.Node{
		html.H2(gomponents.Text("Uptime History")),
		html.Div(
			linkButton("/monitor/history/"+site+"/hour", "hour"),
			linkButton("/monitor/history/"+site+"/day", "day"),
			linkButton("/monitor/history/"+site+"/week", "week"),
			linkButton("/monitor/history/"+site+"/month", "month"),
			linkButton("/", "Home"),
		),
		gomponents.If(history == nil, html.P(gomponents.Text("No data for time period"))),
		historyTable(history),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
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

func notifications(w http.ResponseWriter, r *http.Request) {
	notifications := getAllNotifications()
	rows := []gomponents.Node{}
	for _, n := range notifications {
		row := html.Tr(
			html.Td(gomponents.Text(n.Name)),
			html.Td(gomponents.Text(string(n.Type))),
			html.Td(linkButton("/notifications/edit/"+n.Name, "Edit")),
			html.Td(linkButton("/notifications/test/"+n.Name, "Test")),
			html.Td(formButton("Delete", "/notifications/delete/"+n.Name)),
		)
		rows = append(rows, row)
	}
	if err := layout("Notifications", []gomponents.Node{
		html.H1(gomponents.Text("Notifications")),
		linkButton("/notifications/new", "New Notification"),
		linkButton("/", "Home"),
		html.Br(), html.Br(),
		html.Table(
			html.Tr(
				html.Th(gomponents.Text("Name")),
				html.Th(gomponents.Text("Type")),
				html.Th(gomponents.Text("Actions"), gomponents.Attr("colspan", "3")),
			),
			gomponents.Group(rows),
		),
	}).Render(w); err != nil {
		log.Println("render err", err)
	}
}

func newNotification(w http.ResponseWriter, r *http.Request) {
	if err := layoutExtra("New Notification", []gomponents.Node{
		html.H1(gomponents.Text("New Notifications")),
		html.Form(
			html.Method("post"),
			html.Action("/notifications/new"),
			html.Table(
				inputTableRow("Name", "name", "text", "", "40"),
				radioGroup("Notification Type", "type", []Radio{
					{"slack", "Slack", false},
					{"discord", "Discord", false},
					{"mailgun", "Mailgun", false},
					{"email", "Email", false},
					{"sms", "SMS", false},
				}),
			),
			html.Br(),
			html.Table(
				html.ID("slack"), html.Style("display:none"),
				html.Tr(
					html.Td(gomponents.Text("Slack Data"), gomponents.Attr("colspan", "2")),
				),
				html.Tr(
					html.Td(html.Label(html.For("token"), gomponents.Text("Token"))),
					html.Td(html.Input(html.Name("token"), html.Type("text"), gomponents.Attr("size", "40"))),
				),
				html.Tr(
					html.Td(html.Label(html.For("channel"), gomponents.Text("Channel"))),
					html.Td(html.Input(html.Name("channel"), html.Type("text"), gomponents.Attr("size", "40"))),
				),
			),
			html.Table(
				html.ID("discord"), html.Style("display:none"),
				html.Tr(
					html.Td(gomponents.Text("Discord Data"), gomponents.Attr("colspan", "2")),
				),
				html.Tr(
					html.Td(html.Label(html.For("webhook"), gomponents.Text("Webhook URL"))),
					html.Td(html.Input(html.Name("webhook"), html.Type("text"), gomponents.Attr("size", "40"))),
				),
			),
			html.Table(
				html.ID("mailgun"), html.Style("display:none"),
				html.Tr(
					html.Td(gomponents.Text("MailGun Data"), gomponents.Attr("colspan", "2")),
				),
				html.Tr(
					html.Td(html.Label(html.For("apikey"), gomponents.Text("API Key"))),
					html.Td(html.Input(html.Name("apikey"), html.Type("text"), gomponents.Attr("size", "40"))),
				),
				html.Tr(
					html.Td(html.Label(html.For("domain"), gomponents.Text("Sending Email Domain"))),
					html.Td(html.Input(html.Name("domain"), html.Type("text"), gomponents.Attr("size", "40"))),
				),
				html.Tr(
					html.Td(html.Label(html.For("recipients"), gomponents.Text("Recipient Email Addresses(s)"))),
					html.Td(html.Input(html.Name("recipients"), html.Type("text"),
						gomponents.Attr("size", "40"), html.Multiple())),
				),
			),
			html.Table(
				html.ID("email"), html.Style("display:none"),
				html.Tr(
					html.Td(gomponents.Text("email: coming soon"), gomponents.Attr("colspan", "2")),
				),
			),
			html.Table(
				html.ID("sms"), html.Style("display:none"),
				html.Tr(
					html.Td(gomponents.Text("sms: coming soon"), gomponents.Attr("colspan", "2")),
				),
			),
			html.Br(),
			linkButton("/notifications/", "Cancel"),
			submitButton("Create"),
		),
	}).Render(w); err != nil {
		log.Println("render err", err)
	}
}

func deleletNotification(w http.ResponseWriter, r *http.Request) {
	notify := r.PathValue("notify")
	if err := r.ParseForm(); err != nil {
		displayError(w, err)
		return
	}
	log.Println("delete notifier", notify)
	if err := removeNotify(notify); err != nil {
		displayError(w, err)
		return
	}
	http.Redirect(w, r, "/notifications/", http.StatusFound)
}

func createNotification(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		displayError(w, err)
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
		err = createNotify(slack.Name, Slack, slack)
	case "discord":
		discord := DisordNotifier{
			Name: r.FormValue("name"),
			URL:  r.FormValue("webhook"),
		}
		log.Println("create discord notification", discord)
		err = createNotify(discord.Name, Discord, discord)
	case "mailgun":
		mailgun := MailGunNotifier{
			Name:       r.FormValue("name"),
			APIKey:     r.FormValue("apikey"),
			Domain:     r.FormValue("domain"),
			Recipients: strings.Split(r.FormValue("recipients"), ","),
		}
		log.Println("create mailgun notification", mailgun)
		err = createNotify(mailgun.Name, MailGun, mailgun)
	default:
		err = errors.New("not implemented")
	}
	if err != nil {
		displayError(w, err)
		return
	}
	reset <- syscall.SIGHUP
	http.Redirect(w, r, "/notifications/", http.StatusFound)
}

func displayEditnotification(w http.ResponseWriter, r *http.Request) {
	notify := r.PathValue("notify")
	notifyType, notification, err := getNotify(notify)
	if err != nil {
		displayError(w, err)
		return
	}
	var n Notification
	if err := json.Unmarshal(notification, &n); err != nil {
		displayError(w, err)
		return
	}
	var table, hidden gomponents.Node
	log.Println("....", notifyType, pretty.Sprint(n))
	switch notifyType {
	case Slack:
		var notify SlackNotifier
		if err := json.Unmarshal(notification, &notify); err != nil {
			log.Println("-------------problem", string(notification))
		}
		table = html.Table(
			inputTableRow("Token", "token", "text", notify.Token, "60"),
			inputTableRow("Channel", "channel", "text", notify.Channel, "60"),
		)
		hidden = html.Input(html.Name("type"), html.Type("hidden"), html.Value("slack"))
	case Discord:
		var notify DisordNotifier
		if err := json.Unmarshal(notification, &notify); err != nil {
			log.Println("-------------problem", string(notification))
		}
		table = html.Table(
			inputTableRow("Webhook URL", "webhook", "text", notify.URL, "60"),
		)
		hidden = html.Input(html.Name("type"), html.Type("hidden"), html.Value("discord"))
	case MailGun:
		var notify MailGunNotifier
		if err := json.Unmarshal(notification, &notify); err != nil {
			log.Println("-------------problem", string(notification))
		}
		table = html.Table(
			inputTableRow("API Key", "apikey", "text", notify.APIKey, "60"),
			inputTableRow("Email Domain", "domain", "text", notify.Domain, "60"),
			inputTableRow("Recipient Email(s)", "email", "text", strings.Join(notify.Recipients, ","), "60"),
		)
		hidden = html.Input(html.Name("type"), html.Type("hidden"), html.Value("mailgun"))
	case Email:
		displayError(w, errors.New("notification type not yet implemented"))
		return
	case SMS:
		displayError(w, errors.New("notification type not yet implemented"))
		return
	default:
		displayError(w, errors.New("invalid notification type"))
		return
	}
	if err := layout("Edit Notification", []gomponents.Node{
		html.H1(gomponents.Text("Edit Notification")),
		html.H2(gomponents.Text("Notifications Name: " + n.Name)),
		html.Form(html.Method("post"), html.Action("/notification/edit"),
			table,
			hidden,
			linkButton("/notifications/", "Cancel"),
			submitButton("Update"),
		),
	}).Render(w); err != nil {
		log.Println("render err", err)
	}

}

func editNotification(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		displayError(w, err)
		return
	}
	notificationType := r.FormValue("type")
	switch notificationType {
	case "slack":
		editSlackNotification(w, r)
	case "discord":
		editDiscordNotification(w, r)
	case "mailgun":
		editMailgunNotification(w, r)
	case "email":
		editEmailNotification(w, r)
	case "sms":
		editSMSNotification(w, r)
	default:
		displayError(w, errors.New("invalid notification type"))
		return
	}
}

func testNotification(w http.ResponseWriter, r *http.Request) {
	n := r.PathValue("notify")
	kind, notification, err := getNotify(n)
	if err != nil {
		displayError(w, err)
		return
	}
	switch kind {
	case Slack:
		if err := sendSlackTestNotification(notification); err != nil {
			displayError(w, err)
			return
		}
	case Discord:
		if err := sendDiscordTestNotification(notification); err != nil {
			displayError(w, err)
			return
		}
	case MailGun:
		if err := sendMailGunTestNotification(notification); err != nil {
			displayError(w, err)
			return
		}
	default:
		displayError(w, errors.New("invalid notification type"))
		return
	}
	if err := layout("Test Notification", []gomponents.Node{
		html.H3(gomponents.Text("Text Notfication Sent")),
		html.H4(gomponents.Text(string(kind))),
		linkButton("/notifications/", "Notifications"),
		linkButton("/", "Home"),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func editSMSNotification(w http.ResponseWriter, r *http.Request) {
	if err := layout("Edit Notification", []gomponents.Node{
		html.H2(gomponents.Text("Not Implemented")),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func editEmailNotification(w http.ResponseWriter, r *http.Request) {
	if err := layout("Edit Notification", []gomponents.Node{
		html.H2(gomponents.Text("Not Implemented")),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func editSlackNotification(w http.ResponseWriter, r *http.Request) {
	notification := SlackNotifier{
		Name:    r.PathValue("notify"),
		Token:   r.FormValue("token"),
		Channel: r.FormValue("channel"),
	}
	if err := updateNotify(notification.Name, Slack, notification); err != nil {
		displayError(w, err)
		return
	}
	reset <- syscall.SIGHUP
	http.Redirect(w, r, "/notifications/", http.StatusFound)
}

func editDiscordNotification(w http.ResponseWriter, r *http.Request) {
	notification := DisordNotifier{
		Name: r.PathValue("notify"),
		URL:  r.FormValue("webhook"),
	}
	if err := updateNotify(notification.Name, Discord, notification); err != nil {
		displayError(w, err)
		return
	}
	reset <- syscall.SIGHUP
	http.Redirect(w, r, "/notifications/", http.StatusFound)
}

func editMailgunNotification(w http.ResponseWriter, r *http.Request) {
	notification := MailGunNotifier{
		Name:       r.PathValue("notify"),
		APIKey:     r.FormValue("apikey"),
		Domain:     r.FormValue("domain"),
		Recipients: strings.Split(r.FormValue("recipients"), ","),
	}
	if err := updateNotify(notification.Name, Slack, notification); err != nil {
		displayError(w, err)
		return
	}
	reset <- syscall.SIGHUP
	http.Redirect(w, r, "/notifications/", http.StatusFound)
}
