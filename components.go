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
