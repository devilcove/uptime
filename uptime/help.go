package main

import (
	"fmt"
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type key struct {
	name string
	help string
}

func dialog(p tview.Primitive, w, h int) tview.Primitive { //nolint:ireturn,varnamelen
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, h, 1, true).
			AddItem(nil, 0, 1, false), w, 1, true).
		AddItem(nil, 0, 1, false)
}

func about(p tview.Primitive) tview.Primitive {
	mainKeys := []key{
		{"Esc", "close dialog/application"},
		{"Enter", "site history"},
		{"m", "create new monitor"},
		{"", ""},
		{"?", "detailed help for a pane"},
	}

	table := tview.NewTable()
	for i, key := range mainKeys {
		table.SetCell(i, 0, tview.NewTableCell(key.name).
			SetAlign(tview.AlignCenter).SetExpansion(1).SetTextColor(tcell.ColorLightBlue))
		table.SetCell(i, 1, tview.NewTableCell(key.help).
			SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorBlue))
	}
	app := "Uptime"
	version := fmt.Sprint("Version ", Version)
	copyright := "© 2025 Matthew R Kasun"
	repo := "https://github.com/devilcove/uptime"
	about := tview.NewTable().
		SetCell(1, 0, tview.NewTableCell(app).SetAlign(tview.AlignCenter).SetExpansion(1)).
		SetCell(3, 0, tview.NewTableCell(version).SetAlign(tview.AlignCenter).SetExpansion(1)).
		SetCell(4, 0, tview.NewTableCell(copyright).SetAlign(tview.AlignCenter).SetExpansion(1)).
		SetCell(5, 0, tview.NewTableCell(repo).SetAlign(tview.AlignCenter).SetExpansion(1))
	grid := tview.NewGrid().
		SetRows(6, 0).
		SetColumns(0, 1).
		AddItem(about, 0, 0, 1, 1, 0, 0, false).
		AddItem(table, 9, 0, len(mainKeys), 1, 0, 0, true)
	grid.SetBorder(true)
	_, _, w, _ := p.GetRect()
	width := 0.8 * float64(w)
	height := len(mainKeys) + 10
	return dialog(grid, int(width), int(height))
}

func help(name string, p tview.Primitive, keys []key) tview.Primitive {
	table := tview.NewTable()
	for i, key := range keys {
		table.SetCell(i+1, 0, tview.NewTableCell(key.name).
			SetAlign(tview.AlignCenter).SetExpansion(1).SetTextColor(tcell.ColorLightBlue))
		table.SetCell(i+1, 1, tview.NewTableCell(key.help).
			SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorBlue))
	}
	table.SetBorder(true).SetTitle(name).SetTitleAlign(tview.AlignCenter)
	_, _, w, _ := p.GetRect()
	width := 0.8 * float64(w)
	height := len(keys) + 3
	log.Println("dialog", name, width, height)
	return dialog(table, int(width), height)
}

func errorDialog(msg string) tview.Primitive {
	return tview.NewModal().
		SetText(msg).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			page, _ := pager.GetFrontPage()
			pager.RemovePage(page)
		})
}
