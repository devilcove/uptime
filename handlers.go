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
	"time"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func favicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "files/favicon.svg")
}

func styles(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "files/styles.css")
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	if err := layout("Uptime", []g.Node{
		h.H2(g.Text("Uptime Status")),
		h.Br(nil),
		g.If(IsAdmin(r), linkButton("/monitor/new", "New Monitor")),
		linkButton("notifications/", "Notifications"),
		linkButton("/logs", "View Logs"),
		linkButton("/logout", "Logout"),
		linkButton("/user/", "User Admin"),
		h.Br(nil),
		statusTable(),
	}).Render(w); err != nil {
		log.Println("render main page", err)
	}
}

func logs(w http.ResponseWriter, r *http.Request) {
	logs, err := os.ReadFile("uptime.log")
	if err != nil {
		log.Println("get logs", err)
		http.Error(w, "unable to retrieve logs", http.StatusInternalServerError)
		return
	}
	data := []g.Node{}
	lines := strings.Split(string(logs), "\n")
	for i := len(lines) - 1; i > len(lines)-200; i-- {
		if i < 0 {
			break
		}
		data = append(data, g.Text(lines[i]), h.Br(nil))
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
	if err := layout("Logout", []g.Node{
		h.H2(g.Text("Goodbye")),
		h.Br(nil),
		linkButton("/", "Home"),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func displayLogin(w http.ResponseWriter, r *http.Request) {
	if err := layout("Login", []g.Node{
		h.Form(h.Class("center"),
			h.Action("/login"),
			h.Method("POST"),
			h.Table(
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
	if err := layout("Admin", []g.Node{
		container(true,
			h.H2(g.Text("Admin Page")),
			g.If(data.Admin,
				h.Button(
					g.Attr("type", "button"),
					g.Attr("onclick", "document.getElementById('new').showModal()"),
					g.Text("Create New User"),
				),
			),
			linkButton("/", "Home"),
			h.Br(nil), h.Br(nil),
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

	if err := layout("Edit User", []g.Node{
		h.H2(g.Text("Edit User " + user.Name)),
		h.Form(h.Class("center"),
			h.Action("/user/"+user.Name),
			h.Method("POST"),
			h.Div(
				h.Label(g.Text("Pass")),
				h.Input(
					h.ID("password"),
					h.Type("password"),
					h.Name("pass"),
					g.Attr("size", "40"),
				),
			),
			h.Div(
				h.Label(g.Text("Admin")),
				checkbox("admin", user.Admin),
			),
			h.Div(
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
	if err := layout("Error", []g.Node{
		h.H1(g.Text("An Error Occurred")),
		h.P(g.Text(err.Error())),
		linkButton("/", "Home"),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func newMonitor(w http.ResponseWriter, r *http.Request) {
	notifications := getAllNotifications()
	notifyCheckboxes := make([]g.Node, 0, len(notifications)+1)
	for _, n := range notifications {
		checkbox := h.Input(
			h.Type("checkbox"),
			h.Name(n.Name),
			g.Text(n.Name),
		)
		notifyCheckboxes = append(notifyCheckboxes, checkbox, g.Text(n.Name))
	}
	if err := layout("New Monitor", []g.Node{
		h.H2(g.Text("Create New Monitor")),
		h.Form(
			h.Method("post"),
			h.Action("/monitor/new"),
			h.Table(
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
				h.Tr(
					h.Td(h.Label(g.Text("Notifications"))),
					h.Td(g.Group(notifyCheckboxes)),
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
		Active:  true,
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
		displayError(w, errNotImplemented)
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
	notifyCheckboxes := make([]g.Node, 0, len(notifications)+1)
	for _, n := range notifications {
		checkbox := h.Input(
			h.Type("checkbox"),
			h.Name(n.Name),
			g.Text(n.Name),
			g.If(slices.Contains(monitor.Notifiers, n.Name), h.Checked()),
		)
		notifyCheckboxes = append(notifyCheckboxes, checkbox, g.Text(n.Name))
	}
	if err := layout("Edit Monitor", []g.Node{
		h.H2(g.Text("Edit Monitor")),
		h.Form(
			h.Method("post"),
			h.Action("/monitor/edit/"+monitor.Name),
			h.Table(
				h.Tr(
					h.Td(h.Label(h.For("name"), g.Text("Name"))),
					h.Td(h.Input(
						h.Name("name"),
						h.Type("text"),
						h.Required(),
						h.Value(monitor.Name),
						g.Attr("size", "60"),
					)),
				),
				h.Tr(
					h.Td(h.Label(h.For("url"), g.Text("URL"))),
					h.Td(h.Input(
						h.Name("url"),
						h.Type("text"),
						h.Required(),
						h.Value(monitor.URL),
						g.Attr("size", "60"),
					)),
				),
				h.Tr(
					h.Td(h.Label(h.For("statusok"), g.Text("OK Status"))),
					h.Td(h.Input(
						h.Name("statusok"),
						h.Type("number"),
						h.Required(),
						h.Value("200"),
						g.Attr("size", "60"),
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
				h.Tr(
					h.Td(h.Label(g.Text("Notifications"))),
					h.Td(g.Group(notifyCheckboxes)),
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
	if err := layout("Delete Monitor", []g.Node{
		h.H2(g.Text("Delete Monitor " + monitor)),
		h.Form(
			h.Action("/monitor/delete/"+monitor),
			h.Method("post"),
			h.Input(
				h.Type("checkbox"),
				h.Value("history"),
				h.Name("history"),
			),
			h.Label(
				h.For("history"),
				g.Text("Also deleted history?"),
			),
			h.Br(), h.Br(),
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
	notifications := getAllNotifications()
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
		Active:  true,
	}
	ok, err := strconv.Atoi(r.FormValue("statusok"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// check notifications
	for _, n := range notifications {
		notification := r.FormValue(n.Name)
		if notification == "on" {
			monitor.Notifiers = append(monitor.Notifiers, n.Name)
		}
	}
	monitor.StatusOK = ok
	if monitor.Type == PING {
		displayError(w, errNotImplemented)
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
	case "all":
		timeFrame = All
	default:
		timeFrame = Day
	}
	history, err := getHistory([]string{"history", site}, timeFrame)
	if err != nil {
		log.Println("get status", err)
		http.Error(w, "unable to access database: "+err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(history)
	if err := layout("History", []g.Node{
		h.H2(g.Text("Uptime History")),
		h.Div(
			linkButton("/monitor/history/"+site+"/day", "day"),
			linkButton("/monitor/history/"+site+"/week", "week"),
			linkButton("/monitor/history/"+site+"/month", "month"),
			linkButton("/monitor/history/"+site+"/year", "year"),
			linkButton("/monitor/history/"+site+"/all", "all time"),
			linkButton("/", "Home"),
			g.If(IsAdmin(r), h.Button(h.Type("button"), h.Style("background:red"), g.Text("Purge Data"),
				g.Attr("onclick", "document.getElementById('purge').showModal()"))),
		),
		g.If(history == nil, h.P(g.Text("No data for time period"))),
		h.P(g.Text(strconv.Itoa(len(history)) + " records")),
		historyTable(history),
		histPurgeDialog(site, time.Now().Add(-time.Hour*24*30).Format(time.DateOnly)),
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
	admin := IsAdmin(r)
	notifications := getAllNotifications()
	rows := []g.Node{}
	for _, n := range notifications {
		row := h.Tr(
			h.Td(g.Text(n.Name)),
			h.Td(g.Text(string(n.Type))),
			g.If(admin, h.Td(linkButton("/notifications/edit/"+n.Name, "Edit"))),
			g.If(admin, h.Td(linkButton("/notifications/test/"+n.Name, "Test"))),
			g.If(admin, h.Td(formButton("Delete", "/notifications/delete/"+n.Name))),
		)
		rows = append(rows, row)
	}
	if err := layout("Notifications", []g.Node{
		h.H1(g.Text("Notifications")),
		g.If(admin, linkButton("/notifications/new", "New Notification")),
		linkButton("/", "Home"),
		h.Br(), h.Br(),
		h.Table(
			h.Tr(
				h.Th(g.Text("Name")),
				h.Th(g.Text("Type")),
				g.If(admin, h.Th(g.Text("Actions"), g.Attr("colspan", "3"))),
			),
			g.Group(rows),
		),
	}).Render(w); err != nil {
		log.Println("render err", err)
	}
}

func newNotification(w http.ResponseWriter, r *http.Request) {
	if err := layoutExtra("New Notification", []g.Node{
		h.H1(g.Text("New Notifications")),
		h.Form(
			h.Method("post"),
			h.Action("/notifications/new"),
			h.Table(
				inputTableRow("Name", "name", "text", "", "40"),
				radioGroup("Notification Type", "type", []Radio{
					{"slack", "Slack", false},
					{"discord", "Discord", false},
					{"mailgun", "Mailgun", false},
					{"email", "Email", false},
					{"sms", "SMS", false},
				}),
			),
			h.Br(),
			h.Table(
				h.ID("slack"), h.Style("display:none"),
				h.Tr(
					h.Td(g.Text("Slack Data"), g.Attr("colspan", "2")),
				),
				h.Tr(
					h.Td(h.Label(h.For("token"), g.Text("Token"))),
					h.Td(h.Input(h.Name("token"), h.Type("text"), g.Attr("size", "40"))),
				),
				h.Tr(
					h.Td(h.Label(h.For("channel"), g.Text("Channel"))),
					h.Td(h.Input(h.Name("channel"), h.Type("text"), g.Attr("size", "40"))),
				),
			),
			h.Table(
				h.ID("discord"), h.Style("display:none"),
				h.Tr(
					h.Td(g.Text("Discord Data"), g.Attr("colspan", "2")),
				),
				h.Tr(
					h.Td(h.Label(h.For("webhook"), g.Text("Webhook URL"))),
					h.Td(h.Input(h.Name("webhook"), h.Type("text"), g.Attr("size", "40"))),
				),
			),
			h.Table(
				h.ID("mailgun"), h.Style("display:none"),
				h.Tr(
					h.Td(g.Text("MailGun Data"), g.Attr("colspan", "2")),
				),
				h.Tr(
					h.Td(h.Label(h.For("apikey"), g.Text("API Key"))),
					h.Td(h.Input(h.Name("apikey"), h.Type("text"), g.Attr("size", "40"))),
				),
				h.Tr(
					h.Td(h.Label(h.For("domain"), g.Text("Sending Email Domain"))),
					h.Td(h.Input(h.Name("domain"), h.Type("text"), g.Attr("size", "40"))),
				),
				h.Tr(
					h.Td(h.Label(h.For("recipients"), g.Text("Recipient Email Addresses(s)"))),
					h.Td(h.Input(h.Name("recipients"), h.Type("text"),
						g.Attr("size", "40"), h.Multiple())),
				),
			),
			h.Table(
				h.ID("email"), h.Style("display:none"),
				h.Tr(
					h.Td(g.Text("email: coming soon"), g.Attr("colspan", "2")),
				),
			),
			h.Table(
				h.ID("sms"), h.Style("display:none"),
				h.Tr(
					h.Td(g.Text("sms: coming soon"), g.Attr("colspan", "2")),
				),
			),
			h.Br(),
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
	table, hidden, err := notificationForm(notifyType, notification)
	if err != nil {
		displayError(w, err)
		return
	}
	if err := layout("Edit Notification", []g.Node{
		h.H1(g.Text("Edit Notification")),
		h.H2(g.Text("Notifications Name: " + n.Name)),
		h.Form(h.Method("post"), h.Action("/notifications/edit/"+n.Name),
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
	if err := layout("Test Notification", []g.Node{
		h.H3(g.Text("Text Notfication Sent")),
		h.H4(g.Text(string(kind))),
		linkButton("/notifications/", "Notifications"),
		linkButton("/", "Home"),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func editSMSNotification(w http.ResponseWriter, r *http.Request) { //nolint:unparam
	if err := layout("Edit Notification", []g.Node{
		h.H2(g.Text("Not Implemented")),
	}).Render(w); err != nil {
		log.Println("render error", err)
	}
}

func editEmailNotification(w http.ResponseWriter, r *http.Request) { //nolint:unparam
	if err := layout("Edit Notification", []g.Node{
		h.H2(g.Text("Not Implemented")),
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

func details(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	monitor, err := getMonitor(site)
	if err != nil {
		displayError(w, err)
		return
	}
	history, err := getHistory([]string{"history", site}, All)
	if err != nil {
		displayError(w, err)
		return
	}
	history = compact(history)
	details, err := getHistoryDetails(monitor.Name, monitor.StatusOK)
	if err != nil {
		displayError(w, err)
		return
	}
	var certExpiry, currentResponse g.Node
	if len(history) > 0 {
		currentResponse = h.Td(g.Text(history[0].ResponseTime.Round(time.Millisecond).String()))
		certExpiry = h.Td(g.Text(strconv.Itoa(history[0].CertExpiry) + " days"))
	}
	if err := layout("Details", []g.Node{
		h.H2(g.Text(site)),
		h.P(h.A(h.Href(monitor.URL), g.Text(monitor.URL))),
		h.Div(
			linkButton("/monitor/history/"+site+"/day", "History"),
			g.If(IsAdmin(r),
				g.Group{
					g.If(monitor.Active, linkButton("/monitor/pause/"+site, "Pause")),
					g.If(!monitor.Active, linkButton("/monitor/resume/"+site, "Resume")),
					linkButton("/monitor/edit/"+site, "Edit"),
					linkButton("/monitor/delete/"+site, "Delete"),
				},
			),
			linkButton("/", "Home"),
		),
		h.Br(),
		h.Table(
			h.Tr(
				h.Th(g.Text("Current Response")),
				h.Th(g.Text("24 Hour Avg Response")),
				h.Th(g.Text("30 Day Avg Response")),
				h.Th(g.Text("24 Hour Uptime")),
				h.Th(g.Text("30 Day Uptime")),
				h.Th(g.Text("Certificate Expiry")),
			),
			h.Tr(
				currentResponse,
				h.Td(g.Text(strconv.Itoa(details.Response24)+" ms")),
				h.Td(g.Text(strconv.Itoa(details.Response30)+" ms")),
				h.Td(g.Text(strconv.FormatFloat(details.Uptime24, 'f', 2, 64)+" %")),
				h.Td(g.Text(strconv.FormatFloat(details.Uptime30, 'f', 2, 64)+" %")),
				certExpiry,
			),
		),
		h.Br(),
		compactHistoryTable(history, monitor.StatusOK),
	}).Render(w); err != nil {
		log.Println("render err", err)
	}
}

func pauseMonitor(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	monitor, err := getMonitor(site)
	if err != nil {
		displayError(w, err)
		return
	}
	monitor.Active = false
	if err := saveMonitor(monitor, true); err != nil {
		displayError(w, err)
		return
	}
	reset <- syscall.SIGHUP
	details(w, r)
}

func resumeMonitor(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	monitor, err := getMonitor(site)
	if err != nil {
		displayError(w, err)
		return
	}
	monitor.Active = true
	if err := saveMonitor(monitor, true); err != nil {
		displayError(w, err)
		return
	}
	reset <- syscall.SIGHUP
	details(w, r)
}

func purgeHistory(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	date := r.FormValue("date")
	if err := purgeHistData(site, date); err != nil {
		displayError(w, err)
		return
	}
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
