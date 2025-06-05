package main

import (
	"strconv"
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
	for _, m := range monitors {
		name := h.Button(g.Text(m.Name), h.Style("background:red"), h.Title("Paused"))
		if m.Active {
			name = h.Button(g.Text(m.Name), h.Style("background:green"), h.Title("Active"))
		}
		row := h.Tr(
			h.Td(name),
			h.Td(h.Button(h.Style("background:"+"green"),
				g.Text(strconv.FormatFloat(m.PerCent, 'f', 2, 64)+" %")),
				h.Title("last 24 hours"),
			),
			h.Td(g.Text(m.Status.Time.Format(time.RFC822))),
			h.Td(g.Text(m.Status.ResponseTime.Round(time.Millisecond).String())),
			h.Td(g.Text(strconv.Itoa(m.Status.CertExpiry))),
			h.Td(linkButton("/monitor/details/"+m.Name, "Details")),
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

func compactHistoryTable(history []Status) g.Node {
	rows := []g.Node{}
	header := h.Tr(
		h.Th(g.Text("Status")),
		h.Th(g.Text("Time")),
		h.Th(g.Text("Details")),
	)
	rows = append(rows, header)
	for _, s := range history {
		button := h.Button(g.Attr("style", "background-color:red"), g.Text("Down"))
		if s.StatusCode == 200 {
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
