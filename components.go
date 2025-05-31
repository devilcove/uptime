package main

import (
	"strconv"
	"time"

	"maragu.dev/gomponents"
	"maragu.dev/gomponents/components"
	"maragu.dev/gomponents/html"
)

func container(center bool, children ...gomponents.Node) gomponents.Node {
	style := ""
	if center {
		style = "text-align:center; margin-left:auto; margin-right:auto"
	}
	return html.Div(
		html.Style(style),
		gomponents.Group(children),
	)
}

func linkButton(href, text string) gomponents.Node {
	return html.Button(
		html.Type("button"),
		gomponents.Attr("onclick", "goTo('"+href+"')"),
		gomponents.Text(text),
	)
}

func submitButton(name string) gomponents.Node {
	return html.Button(
		html.Type("submit"),
		gomponents.Text(name),
	)
}

func formButton(name, action string) gomponents.Node {
	return html.Form(
		html.Action(action),
		html.Method("post"),
		gomponents.Attr("onsubmit", "return confirm('confirm')"),
		html.Button(
			html.Type("submit"),
			gomponents.Text(name),
		),
	)
}

func checkbox(name string, checked bool) gomponents.Node {
	return html.Input(
		html.Type("checkbox"),
		html.Name(name),
		gomponents.If(checked, gomponents.Attr("checked", "true")),
	)
}

func navbar(authenticated bool, currentPath string) gomponents.Node {
	return html.Nav(
		navbarLink("/", "Home", currentPath),
		navbarLink("/about", "About", currentPath),
		gomponents.If(authenticated, navbarLink("/profile", "Profile", currentPath)),
	)
}

func navbarLink(href, name, currentPath string) gomponents.Node {
	return html.A(html.Href(href), components.Classes{"is-active": currentPath == href}, gomponents.Text(name))
}

func inputTableRow(label, name, kind, value, size string) gomponents.Node {
	return html.Tr(
		html.Td(html.Label(html.For(name), gomponents.Text(label))),
		html.Td(html.Input(
			html.Name(name), html.ID(name), html.Type(kind), html.Value(value),
			gomponents.Attr("size", size), gomponents.Text(name))),
	)
}

func userTable(users []User) gomponents.Node {
	rows := []gomponents.Node{}
	header := html.Tr(
		html.Th(gomponents.Text("Name")),
		html.Th(gomponents.Attr("colspan=\"2\""), gomponents.Text("Actions")),
	)
	rows = append(rows, header)
	for _, user := range users {
		row := html.Tr(
			html.Td(html.Label(gomponents.Text(user.Name))),
			html.Td(linkButton("/user/"+user.Name, "Edit")),
			html.Td(formButton("Delete", "/user/delete/"+user.Name)),
		)
		rows = append(rows, row)
	}
	return html.Table(
		gomponents.Group(rows),
	)
}

func statusTable(status []Status) gomponents.Node {
	rows := []gomponents.Node{}
	header := html.Tr(
		html.Th(gomponents.Text("Site")),
		html.Th(gomponents.Text("Status")),
		html.Th(gomponents.Text("Code")),
		html.Th(gomponents.Text("Time")),
		html.Th(gomponents.Text("Response Time")),
		html.Th(gomponents.Text("Cert Expiry")),
		html.Th(gomponents.Attr("colspan=\"2\""), gomponents.Text("Actions")),
	)
	rows = append(rows, header)
	for _, s := range status {
		link := "/monitor/history/" + s.Site + "/day"
		deleteLink := "/monitor/delete/" + s.Site
		editLink := "/monitor/edit/" + s.Site
		row := html.Tr(
			html.Td(html.A(html.Href(link), gomponents.Text(s.Site))),
			html.Td(gomponents.Text(s.Status)),
			html.Td(gomponents.Text(strconv.Itoa(s.StatusCode))),
			html.Td(gomponents.Text(s.Time.Local().Format(time.RFC822))),
			html.Td(gomponents.Text(s.ResponseTime.Round(time.Millisecond).String())),
			html.Td(gomponents.Text(strconv.Itoa(s.CertExpiry))),
			html.Td(linkButton(editLink, "Edit")),
			html.Td(linkButton(deleteLink, "Delete")),
		)
		rows = append(rows, row)
	}
	return html.Table(
		gomponents.Group(rows),
	)
}

func historyTable(history []Status) gomponents.Node {
	rows := []gomponents.Node{}
	header := html.Tr(
		html.Th(gomponents.Text("Site")),
		html.Th(gomponents.Text("Status")),
		html.Th(gomponents.Text("Code")),
		html.Th(gomponents.Text("Time")),
		html.Th(gomponents.Text("Response Time")),
		html.Th(gomponents.Text("Cert Expiry")),
	)
	rows = append(rows, header)
	for _, s := range history {
		row := html.Tr(
			html.Td(gomponents.Text(s.Site)),
			html.Td(gomponents.Text(s.Status)),
			html.Td(gomponents.Text(strconv.Itoa(s.StatusCode))),
			html.Td(gomponents.Text(s.Time.Local().Format(time.RFC822))),
			html.Td(gomponents.Text(s.ResponseTime.Round(time.Millisecond).String())),
			html.Td(gomponents.Text(strconv.Itoa(s.CertExpiry))),
		)
		rows = append(rows, row)
	}
	return html.Table(
		gomponents.Group(rows),
	)
}

func newUserDialog() gomponents.Node {
	return html.Dialog(
		gomponents.Attr("id", "new"),
		html.H2(gomponents.Text("New User")),
		html.Form(
			html.Method("post"),
			html.Action("/user/add"),
			html.Table(
				inputTableRow("Name", "name", "text", "", "40"),
				inputTableRow("Pass", "pass", "password", "", "40"),
			),
			html.Label(html.For("showpass"), gomponents.Text("Show Password"), gomponents.Attr("onclick", "togglePass();")),
			html.Br(),
			html.Label(
				gomponents.Attr("for", "admin"),
				gomponents.Text("Admin"),
			),
			html.Input(
				gomponents.Attr("name", "admin"),
				gomponents.Attr("type", "checkbox"),
			),
			html.Br(),
			html.Br(),
			html.Button(
				gomponents.Attr("type", "button"),
				gomponents.Attr("onclick", "document.getElementById('new').close()"),
				gomponents.Text("Cancel"),
			),
			submitButton("Create"),
		),
		html.Script(gomponents.Raw(
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

func radioGroup(label, name string, radios []Radio) gomponents.Node {
	inputs := []gomponents.Node{}
	for _, radio := range radios {
		input := html.Input(
			html.Name(name),
			html.ID(name),
			html.Type("radio"),
			html.Required(),
			html.Value(radio.Value),
			gomponents.If(radio.Checked, html.Checked()),
			gomponents.Attr("onclick", "displayExtra('"+radio.Value+"')"),
		)
		inputs = append(inputs, input)
		inputs = append(inputs, gomponents.Text(radio.Label))
	}
	return html.Tr(
		html.Td(html.Label(html.For(name), gomponents.Text(label))),
		html.Td(gomponents.Group(inputs)),
	)
}

func hiddenDiv(id, header string, data [][]string) []gomponents.Node {
	rows := []gomponents.Node{}
	for _, x := range data {
		row := []gomponents.Node{
			html.Br(),
			html.Label(html.For(x[0]), gomponents.Text(x[1])),
			html.Input(html.Name(x[0]), html.Type("text"), gomponents.Attr("size", "40")),
		}
		rows = append(rows, row...)
	}
	return rows
}

func togglePass(id string) gomponents.Group {
	return gomponents.Group{
		html.Label(html.For("showpass"), gomponents.Text("Show"), gomponents.Attr("onclick", "togglePass("+id+")")),
		html.Script(gomponents.Raw(
			`function togglePass(id){
				var x = document.getElementById(id);
				if (x.type == "password") {
					x.type = "text";
				} else {
					x.type = "password"; 
				}
			}`)),
	}
}
