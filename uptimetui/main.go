package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/term"
)

var (
	app     *tview.Application
	pager   *tview.Pages
	Version = "v0.1.0"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logFile, err := os.Create("uptime.log")
	if err == nil {
		log.SetOutput(logFile)
	}
	flag.Parse()
	w, h, err := term.GetSize(0)
	log.Println("terminal size", w, h)
	if err != nil {
		log.Fatal("not a term", err)
		fmt.Println("not a term", err)
		os.Exit(1)
	}
	if w < 60 || h < 20 {
		fmt.Println("terminal is too small")
		log.Fatal("terminal is too small", w, h)
		os.Exit(1)
	}
	//header := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("Header")
	footer := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("? for help; esc to exit")
	table := newTable()
	grid := tview.NewGrid().
		SetRows(0, 1).
		//SetBorders(true).
		//AddItem(header, 0, 0, 1, 1, 0, 0, false).
		AddItem(table, 0, 0, 1, 1, 0, 0, true).
		AddItem(footer, 1, 0, 1, 1, 0, 0, false)
	grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		log.Println("grid key handler", event.Name())
		return event
	})
	grid.SetBorder(true).SetTitle("Uptime").SetTitleAlign(tview.AlignCenter)
	pager = tview.NewPages().AddPage("main", grid, true, true)
	pager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		log.Println("pager key handler", event.Name())
		if event.Key() == tcell.KeyEsc {
			front, _ := pager.GetFrontPage()
			if front != "main" {
				pager.RemovePage(front)
				return nil
			}
			app.Stop()
		}
		return event
	})
	app = tview.NewApplication().SetRoot(pager, true).EnableMouse(true)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		log.Println("app key handler", event.Name())
		switch event.Key() {
		case tcell.KeyCtrlR:
			log.Println("restart daemon")
			signalDaemon()
		case tcell.KeyEnter:
			log.Println("enter key")
		case tcell.KeyEsc:
			//log.Println("shutting down")
			//app.Stop()
			//return nil
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
