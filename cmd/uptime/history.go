package main

import (
	"fmt"
	"log"
	"time"

	"github.com/devilcove/uptime"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func history(site string, timeframe uptime.TimeFrame) tview.Primitive {
	historyKeys := []key{
		{"d", "show history for today"},
		{"h", "show history for last hour"},
		{"w", "show history for week"},
		{"m", "show history for month"},
		{"y", "show histor for year"},
		{"", ""},
		{"esc", "close history"},
	}
	log.Println("get history for", site, timeframe.Name())
	path := []string{"history", site}

	db, err := uptime.OpenDB()
	if err != nil {
		log.Println("database open", err)
		return nil
	}
	defer db.Close()
	hist, err := uptime.GetHistory(db, path, timeframe)
	if err != nil {
		log.Println("get history", err)
		return nil
	}
	log.Println("history entries", len(hist))
	headings := []string{"Site", "Status", "Code", "Time"}
	table := tview.NewTable().SetSelectable(true, false).SetSelectedStyle(tcell.StyleDefault)
	for i, heading := range headings {
		table.SetCell(0, i, tview.NewTableCell(heading).
			SetAlign(tview.AlignCenter).SetExpansion(1).SetSelectable(false))
	}
	for i, row := range hist {
		table.SetCell(i+2, 0, tview.NewTableCell(row.Site).
			SetAlign(tview.AlignCenter).SetExpansion(1).SetReference(row.Site))
		table.SetCell(i+2, 1, tview.NewTableCell(row.Status).
			SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(i+2, 2, tview.NewTableCell(fmt.Sprintf("%d", row.StatusCode)).
			SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(i+2, 3, tview.NewTableCell(row.Time.Format(time.RFC822)).
			SetAlign(tview.AlignCenter).SetExpansion(1))
	}
	table.SetBorder(true).SetTitle("History").SetTitleAlign(tview.AlignCenter)

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'd':
				page := history(table.GetCell(2, 0).GetReference().(string), uptime.Day)
				pager.AddPage("history", page, true, true)
			case 'h':
				page := history(table.GetCell(2, 0).GetReference().(string), uptime.Hour)
				pager.AddPage("history", page, true, true)
			case 'm':
				page := history(table.GetCell(2, 0).GetReference().(string), uptime.Month)
				pager.AddPage("history", page, true, true)
			case 'w':
				page := history(table.GetCell(2, 0).GetReference().(string), uptime.Week)
				pager.AddPage("history", page, true, true)
			case 'y':
				page := history(table.GetCell(2, 0).GetReference().(string), uptime.Year)
				pager.AddPage("history", page, true, true)
			case '?':
				help := help("history help", table, historyKeys)
				pager.AddPage("help", help, true, true)
			}
		}
		return event
	})

	return table
}
