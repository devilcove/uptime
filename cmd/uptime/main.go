package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	app *tview.Application
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logFile, err := os.Create("uptime.log")
	if err == nil {
		log.SetOutput(logFile)
	}
	flag.Parse()
	header := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("Header")
	footer := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("footer")
	table := newTable()
	grid := tview.NewGrid().
		SetRows(1, 0, 1).
		SetBorders(true).
		AddItem(header, 0, 0, 1, 1, 0, 0, false).
		AddItem(table, 1, 0, 1, 1, 0, 0, true).
		AddItem(footer, 2, 0, 1, 1, 0, 0, false)
	grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		log.Println("grid key handler", event.Name())
		return event
	})
	pager := tview.NewPages().AddPage("main", grid, true, true)
	pager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		log.Println("pager key handler", event.Name())
		return event
	})
	app = tview.NewApplication().SetRoot(pager, true).EnableMouse(true)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		log.Println("app key handler", event.Name())
		switch event.Key() {
		case tcell.KeyEnter:
			log.Println("enter key")
		case tcell.KeyEsc:
			log.Println("shutting down")
			app.Stop()
			return nil
		case tcell.KeyCtrlC:
			return nil
		}
		return event
	})
	//monitor(table)
	go func() {
		timer := time.NewTimer(time.Second)
		ticker := time.NewTicker(time.Minute)
		//defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				monitor(table)
			case <-timer.C:
				monitor(table)
			}
		}
	}()
	if err := app.Run(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

}
