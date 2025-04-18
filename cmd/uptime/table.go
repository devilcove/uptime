package main

import (
	"fmt"
	"log"
	"time"

	"github.com/devilcove/uptime"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newTable() *tview.Table {
	headings := []string{"Site", "Status", "Code", "Time"}
	table := tview.NewTable().SetSelectable(true, false).SetSelectedStyle(tcell.StyleDefault)
	for i, heading := range headings {
		table.SetCell(0, i, tview.NewTableCell(heading).
			SetAlign(tview.AlignCenter).SetExpansion(1))
	}
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		log.Println("table key handler", tcell.KeyNames[event.Key()])
		return event
	})
	return table
}

// updatetable refresh table with latest data.  do not call from main, only from goroutine
func updateTable(table *tview.Table, data []uptime.Status) {
	for i, row := range data {
		log.Println("updating table row", i)
		table.SetCell(i+2, 0, tview.NewTableCell(row.Site).
			SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(i+2, 1, tview.NewTableCell(row.Status).
			SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(i+2, 2, tview.NewTableCell(fmt.Sprintf("%d", row.StatusCode)).
			SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(i+2, 3, tview.NewTableCell(row.Time.Format(time.RFC822)).
			SetAlign(tview.AlignCenter).SetExpansion(1))
	}
	app.Draw()
}
