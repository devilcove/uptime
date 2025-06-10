package main

import (
	"maragu.dev/gomponents"
	"maragu.dev/gomponents/components"
	"maragu.dev/gomponents/html"
)

func layout(title string, nodes []gomponents.Node) gomponents.Node {
	return components.HTML5(components.HTML5Props{
		Title:    title,
		Language: "en",
		Head: []gomponents.Node{
			html.Link(html.Rel("stylesheet"), html.Href("/styles.css")),
			html.Link(html.Rel("icon"), html.Href("/favicon.ico"), html.Type("image/svg")),
			html.Script(gomponents.Text("function goTo(loc) { location.href=loc }")),
		},
		Body: []gomponents.Node{
			html.Style("background-color: #cecece;"),
			container(true, nodes...),
		},
	})
}

func layoutExtra(title string, nodes []gomponents.Node) gomponents.Node {
	return components.HTML5(components.HTML5Props{
		Title:    title,
		Language: "en",
		Head: []gomponents.Node{
			html.Link(html.Rel("stylesheet"), html.Href("/styles.css")),
			html.Link(html.Rel("icon"), html.Href("/favicon.ico"), html.Type("image/svg")),
			html.Script(
				gomponents.Raw(`function displayExtra(id) {
				document.getElementById('slack').style.display = "none";
				document.getElementById('email').style.display = "none";
				document.getElementById('mailgun').style.display = "none";
				document.getElementById('discord').style.display = "none";
				document.getElementById('sms').style.display = "none";
				document.getElementById(id).style.display = "inline-table";}`),
			),
			html.Script(
				gomponents.Text("function goTo(loc) { location.href=loc }"),
			),
		},
		Body: []gomponents.Node{
			html.Style("background-color: #cecece;"),
			container(true, nodes...),
		},
	})
}

func displayLogs(nodes []gomponents.Node) gomponents.Node {
	return components.HTML5(components.HTML5Props{
		Title:    "Logs",
		Language: "en",
		Head: []gomponents.Node{
			html.Script(gomponents.Text("function goTo(loc) { location.href=loc }")),
		},
		Body: []gomponents.Node{
			html.Style("background-color: #cecece;"),
			container(true,
				linkButton("/logs", "Refresh"),
				linkButton("/", "Home"),
			),
			container(false, nodes...),
		},
	},
	)
}
