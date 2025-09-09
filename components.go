package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func container(center bool, children ...g.Node) g.Node {
	style := ""
	if center {
		style = "text-align:center; margin-left:auto; margin-right:auto"
	}
	return h.Div(
		h.Style(style),
		g.Group(children),
	)
}

func linkButton(href, text string) g.Node {
	return h.Button(
		h.Type("button"),
		g.Attr("onclick", "goTo('"+href+"')"),
		g.Text(text),
		h.Title(href),
	)
}

func submitButton(name string) g.Node {
	return h.Button(
		h.Type("submit"),
		g.Text(name),
	)
}

func formButton(name, action string) g.Node {
	return h.Form(
		h.Action(action),
		h.Method("post"),
		g.Attr("onsubmit", "return confirm('confirm')"),
		h.Button(
			h.Type("submit"),
			g.Text(name),
		),
	)
}

func checkbox(name string, checked bool) g.Node {
	return h.Input(
		h.Type("checkbox"),
		h.Name(name),
		g.If(checked, g.Attr("checked", "true")),
	)
}

func inputTableRow(label, name, kind, value, size string) g.Node {
	return h.Tr(
		h.Td(h.Label(h.For(name), g.Text(label))),
		h.Td(h.Input(
			h.Name(name), h.ID(name), h.Type(kind), h.Value(value),
			g.Attr("size", size), g.Text(name))),
	)
}

func userTable(users []User) g.Node {
	rows := []g.Node{}
	header := h.Tr(
		h.Th(g.Text("Name")),
		h.Th(g.Attr("colspan=\"2\""), g.Text("Actions")),
	)
	rows = append(rows, header)
	for _, user := range users {
		row := h.Tr(
			h.Td(h.Label(g.Text(user.Name))),
			h.Td(linkButton("/user/"+user.Name, "Edit")),
			h.Td(formButton("Delete", "/user/delete/"+user.Name)),
		)
		rows = append(rows, row)
	}
	return h.Table(
		g.Group(rows),
	)
}

func statusTable() g.Node {
	monitors := getAllMonitorsForDisplay()
	rows := []g.Node{}
	header := h.Tr(
		h.Th(g.Text("Monitor")),
		h.Th(g.Text("Uptime")),
		h.Th(g.Text("Time")),
		h.Th(g.Text("Response Time")),
		h.Th(g.Text("Cert Expiry")),
		h.Th(g.Text("Actions")),
	)
	rows = append(rows, header)
	for _, monitor := range monitors {
		name := h.Button(g.Text(monitor.Name), h.Style("background:yellow;color:black;"), h.Title("Paused"))
		if monitor.Active {
			name = h.Button(g.Text(monitor.Name), h.Style("background:red"), h.Title("Down"))
			if monitor.DisplayStatus {
				name = h.Button(g.Text(monitor.Name), h.Style("background:green"), h.Title("Active"))
			}
		}
		row := h.Tr(
			h.Td(name),
			h.Td(h.Button(h.Style("background:"+"green"),
				g.Text(strconv.FormatFloat(monitor.PerCent, 'f', 2, 64)+" %")),
				h.Title("last 24 hours"),
			),
			h.Td(g.Text(monitor.Status.Time.Format(time.RFC822))),
			h.Td(g.Text(monitor.Status.ResponseTime.Round(time.Millisecond).String())),
			h.Td(g.Text(strconv.Itoa(monitor.Status.CertExpiry))),
			h.Td(linkButton("/monitor/details/"+monitor.Name, "Details")),
		)
		rows = append(rows, row)
	}

	return h.Table(
		g.Group(rows),
	)
}

func historyTable(history []Status) g.Node {
	rows := []g.Node{}
	header := h.Tr(
		h.Th(g.Text("Site")),
		h.Th(g.Text("Status")),
		h.Th(g.Text("Code")),
		h.Th(g.Text("Time")),
		h.Th(g.Text("Response Time")),
		h.Th(g.Text("Cert Expiry")),
	)
	rows = append(rows, header)
	for _, s := range history {
		row := h.Tr(
			h.Td(g.Text(s.Site)),
			h.Td(g.Text(s.Status)),
			h.Td(g.Text(strconv.Itoa(s.StatusCode))),
			h.Td(g.Text(s.Time.Local().Format(time.RFC822))),
			h.Td(g.Text(s.ResponseTime.Round(time.Millisecond).String())),
			h.Td(g.Text(strconv.Itoa(s.CertExpiry))),
		)
		rows = append(rows, row)
	}
	return h.Table(
		g.Group(rows),
	)
}

func compactHistoryTable(history []Status, statusOK int) g.Node {
	rows := []g.Node{}
	header := h.Tr(
		h.Th(g.Text("Status")),
		h.Th(g.Text("Time")),
		h.Th(g.Text("Details")),
	)
	rows = append(rows, header)
	for _, s := range history {
		button := h.Button(g.Attr("style", "background-color:red"), g.Text("Down"))
		if s.StatusCode == statusOK {
			button = h.Button(g.Attr("style", "background-color:green"), g.Text("Up"))
		}
		row := h.Tr(
			h.Td(button),
			h.Td(g.Text(s.Time.Local().Format(time.RFC822))),
			h.Td(g.Text(s.Status)),
		)
		rows = append(rows, row)
	}
	return h.Table(
		g.Group(rows),
	)
}

func newUserDialog() g.Node {
	return h.Dialog(
		h.Style("background-color: #4a4a4a; color: white"),
		g.Attr("id", "new"),
		h.H2(g.Text("New User")),
		h.Form(
			h.Method("post"),
			h.Action("/user/add"),
			h.Table(
				inputTableRow("Name", "name", "text", "", "40"),
				inputTableRow("Pass", "pass", "password", "", "40"),
			),
			h.Label(h.For("showpass"), g.Text("Show Password"), g.Attr("onclick", "togglePass();")),
			h.Br(),
			h.Label(
				g.Attr("for", "admin"),
				g.Text("Admin"),
			),
			h.Input(
				g.Attr("name", "admin"),
				g.Attr("type", "checkbox"),
			),
			h.Br(),
			h.Br(),
			h.Button(
				g.Attr("type", "button"),
				g.Attr("onclick", "document.getElementById('new').close()"),
				g.Text("Cancel"),
			),
			submitButton("Create"),
		),
		h.Script(g.Raw(
			`function togglePass() {
				var x = document.getElementById('pass');
				if (x.type == "password"){
					x.type = "text";
				} else {
					x.type = "password"; 
				}
			}`)),
	)
}

func histPurgeDialog(site, value string) g.Node {
	return h.Dialog(
		h.Style("background-color: #4a4a4a; color: white"),
		g.Attr("id", "purge"),
		h.H2(g.Text("Purge Records")),
		h.Form(
			h.Method("post"),
			h.Action("/monitor/history/purge/"+site),
			h.Label(h.For("date"), g.Text("Purge all record before:")),
			h.Br(),
			h.Input(h.Name("date"), h.Type("date"), g.Attr("value", value)),
			h.Br(), h.Br(),
			h.Button(h.Type("button"), g.Text("Cancel"), h.Style("background:red"),
				g.Attr("onclick", "document.getElementById('purge').close()")),
			submitButton("Purge"),
		),
	)
}

func radioGroup(label, name string, radios []Radio) g.Node {
	inputs := []g.Node{}
	for _, radio := range radios {
		input := h.Input(
			h.Name(name),
			h.ID(name),
			h.Type("radio"),
			h.Required(),
			h.Value(radio.Value),
			g.If(radio.Checked, h.Checked()),
			g.Attr("onclick", "displayExtra('"+radio.Value+"')"),
		)
		inputs = append(inputs, input)
		inputs = append(inputs, g.Text(radio.Label))
	}
	return h.Tr(
		h.Td(h.Label(h.For(name), g.Text(label))),
		h.Td(g.Group(inputs)),
	)
}

func notificationForm(kind NotifyType, notification []byte) (g.Node, g.Node, error) {
	switch kind {
	case Slack:
		var n SlackNotifier
		if err := json.Unmarshal(notification, &n); err != nil {
			return nil, nil, fmt.Errorf("invalid slack payload: %w", err)
		}
		return h.Table(
				inputTableRow("Token", "token", "text", n.Token, "60"),
				inputTableRow("Channel", "channel", "text", n.Channel, "60"),
			),
			h.Input(h.Name("type"), h.Type("hidden"), h.Value("slack")),
			nil

	case Discord:
		var n DisordNotifier
		if err := json.Unmarshal(notification, &n); err != nil {
			return nil, nil, fmt.Errorf("invalid discord payload: %w", err)
		}
		return h.Table(
				inputTableRow("Webhook URL", "webhook", "text", n.URL, "60"),
			),
			h.Input(h.Name("type"), h.Type("hidden"), h.Value("discord")),
			nil

	case MailGun:
		var n MailGunNotifier
		if err := json.Unmarshal(notification, &n); err != nil {
			return nil, nil, fmt.Errorf("invalid mailgun payload: %w", err)
		}
		return h.Table(
				inputTableRow("API Key", "apikey", "text", n.APIKey, "60"),
				inputTableRow("Email Domain", "domain", "text", n.Domain, "60"),
				inputTableRow("Recipient Email(s)", "email", "text",
					strings.Join(n.Recipients, ","), "60"),
			),
			h.Input(h.Name("type"), h.Type("hidden"), h.Value("mailgun")),
			nil

	case Email, SMS:
		return nil, nil, errors.New("notification type not yet implemented")

	default:
		return nil, nil, errors.New("invalid notification type")
	}
}

func aboutDialog() g.Node {
	return h.Dialog(
		h.Style("background-color: #4a4a4a; color: white"),
		g.Attr("id", "about"),
		h.H2(g.Text("Uptime")),
		h.P(g.Text("Version v0.1.3")),
		h.H3(g.Text("© 2025 Matthew R Kasun")),
		h.P(
			h.A(
				h.Href("mailto://mkasun@nusak.ca?subject=uptime"),
				envelopeSVG(),
				g.Text(" mkasun@nusak.ca"),
			),
		),
		h.P(
			h.A(
				h.Href("https://github.com/devilcove/uptime"),
				githubSVG(),
				g.Text(" repo"),
			),
		),
		h.Button(
			g.Attr("type", "button"),
			g.Attr("onclick", "document.getElementById('about').close()"),
			g.Text("Close"),
		),
	)
}

func githubSVG() g.Node {
	return h.SVG(
		g.Attr("width", "25"),
		g.Attr("height", "24"),
		g.Attr("viewbox", "0 0 100 100"),
		g.Attr("fill", "currentColor"),
		g.El("path",
			g.Attr("fill-rule", "evenodd"),
			g.Attr("d", "M48.854 0C21.839 0 0 22 0 49.217c0 21.756 13.993 40.172 33.405 46.69 2.427.49 3.316-1.059 3.316-2.362 0-1.141-.08-5.052-.08-9.127-13.59 2.934-16.42-5.867-16.42-5.867-2.184-5.704-5.42-7.17-5.42-7.17-4.448-3.015.324-3.015.324-3.015 4.934.326 7.523 5.052 7.523 5.052 4.367 7.496 11.404 5.378 14.235 4.074.404-3.178 1.699-5.378 3.074-6.6-10.839-1.141-22.243-5.378-22.243-24.283 0-5.378 1.94-9.778 5.014-13.2-.485-1.222-2.184-6.275.486-13.038 0 0 4.125-1.304 13.426 5.052a46.97 46.97 0 0 1 12.214-1.63c4.125 0 8.33.571 12.213 1.63 9.302-6.356 13.427-5.052 13.427-5.052 2.67 6.763.97 11.816.485 13.038 3.155 3.422 5.015 7.822 5.015 13.2 0 18.905-11.404 23.06-22.324 24.283 1.78 1.548 3.316 4.481 3.316 9.126 0 6.6-.08 11.897-.08 13.526 0 1.304.89 2.853 3.316 2.364 19.412-6.52 33.405-24.935 33.405-46.691C97.707 22 75.788 0 48.854 0z"),
			g.Attr("clip-rule", "evenodd"),
		),
	)
}

func envelopeSVG() g.Node {
	return h.SVG(
		g.Attr("width", "25"),
		g.Attr("height", "25"),
		g.Attr("viewbox", "0 -150 640 640"),
		g.Attr("fill", "currentColor"),
		g.El("path",
			g.Attr("d", "M112 128C85.5 128 64 149.5 64 176C64 191.1 71.1 205.3 83.2 214.4L291.2 370.4C308.3 383.2 331.7 383.2 348.8 370.4L556.8 214.4C568.9 205.3 576 191.1 576 176C576 149.5 554.5 128 528 128L112 128zM64 260L64 448C64 483.3 92.7 512 128 512L512 512C547.3 512 576 483.3 576 448L576 260L377.6 408.8C343.5 434.4 296.5 434.4 262.4 408.8L64 260z"),
		),
	)
}
