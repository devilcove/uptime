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
	path := "M237.9 461.4C237.9 463.4 235.6 465 232.7 465C229.4 465.3 227.1 463.7 227.1 461.4C227.1 459.4 229.4 457.8 232.3 457.8C235.3 457.5 237.9 459.1 237.9 461.4zM206.8 456.9C206.1 458.9 208.1 461.2 211.1 461.8C213.7 462.8 216.7 461.8 217.3 459.8C217.9 457.8 216 455.5 213 454.6C210.4 453.9 207.5 454.9 206.8 456.9zM251 455.2C248.1 455.9 246.1 457.8 246.4 460.1C246.7 462.1 249.3 463.4 252.3 462.7C255.2 462 257.2 460.1 256.9 458.1C256.6 456.2 253.9 454.9 251 455.2zM316.8 72C178.1 72 72 177.3 72 316C72 426.9 141.8 521.8 241.5 555.2C254.3 557.5 258.8 549.6 258.8 543.1C258.8 536.9 258.5 502.7 258.5 481.7C258.5 481.7 188.5 496.7 173.8 451.9C173.8 451.9 162.4 422.8 146 415.3C146 415.3 123.1 399.6 147.6 399.9C147.6 399.9 172.5 401.9 186.2 425.7C208.1 464.3 244.8 453.2 259.1 446.6C261.4 430.6 267.9 419.5 275.1 412.9C219.2 406.7 162.8 398.6 162.8 302.4C162.8 274.9 170.4 261.1 186.4 243.5C183.8 237 175.3 210.2 189 175.6C209.9 169.1 258 202.6 258 202.6C278 197 299.5 194.1 320.8 194.1C342.1 194.1 363.6 197 383.6 202.6C383.6 202.6 431.7 169 452.6 175.6C466.3 210.3 457.8 237 455.2 243.5C471.2 261.2 481 275 481 302.4C481 398.9 422.1 406.6 366.2 412.9C375.4 420.8 383.2 435.8 383.2 459.3C383.2 493 382.9 534.7 382.9 542.9C382.9 549.4 387.5 557.3 400.2 555C500.2 521.8 568 426.9 568 316C568 177.3 455.5 72 316.8 72zM169.2 416.9C167.9 417.9 168.2 420.2 169.9 422.1C171.5 423.7 173.8 424.4 175.1 423.1C176.4 422.1 176.1 419.8 174.4 417.9C172.8 416.3 170.5 415.6 169.2 416.9zM158.4 408.8C157.7 410.1 158.7 411.7 160.7 412.7C162.3 413.7 164.3 413.4 165 412C165.7 410.7 164.7 409.1 162.7 408.1C160.7 407.5 159.1 407.8 158.4 408.8zM190.8 444.4C189.2 445.7 189.8 448.7 192.1 450.6C194.4 452.9 197.3 453.2 198.6 451.6C199.9 450.3 199.3 447.3 197.3 445.4C195.1 443.1 192.1 442.8 190.8 444.4zM179.4 429.7C177.8 430.7 177.8 433.3 179.4 435.6C181 437.9 183.7 438.9 185 437.9C186.6 436.6 186.6 434 185 431.7C183.6 429.4 181 428.4 179.4 429.7z" //nolint:lll
	return h.SVG(
		g.Attr("width", "25"),
		g.Attr("height", "24"),
		g.Attr("viewbox", "0 -76 640 640"),
		g.Attr("fill", "currentColor"),
		g.El("path", g.Attr("d", path)),
	)
}

func envelopeSVG() g.Node {
	path := "M112 128C85.5 128 64 149.5 64 176C64 191.1 71.1 205.3 83.2 214.4L291.2 370.4C308.3 383.2 331.7 383.2 348.8 370.4L556.8 214.4C568.9 205.3 576 191.1 576 176C576 149.5 554.5 128 528 128L112 128zM64 260L64 448C64 483.3 92.7 512 128 512L512 512C547.3 512 576 483.3 576 448L576 260L377.6 408.8C343.5 434.4 296.5 434.4 262.4 408.8L64 260z" //nolint:lll
	return h.SVG(
		g.Attr("width", "25"),
		g.Attr("height", "25"),
		g.Attr("viewbox", "0 -150 640 640"),
		g.Attr("fill", "currentColor"),
		g.El("path", g.Attr("d", path)),
	)
}
