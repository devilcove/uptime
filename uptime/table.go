package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
			SetAlign(tview.AlignCenter).SetExpansion(1).SetSelectable(false))
	}
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		log.Println("table key handler", tcell.KeyNames[event.Key()])
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			//case 'l':
			//logs := dialog(showLogs("logs"), 600, 400)
			//pager.AddPage("logs", logs, true, true)
			case 'm':
				monitor := monitorForm("monitor")
				pager.AddPage("monitor", monitor, true, true)
			case '?':
				about := about(table)
				pager.AddPage("help", about, true, true)
				return nil
			}
		}
		return event
	})
	table.SetSelectedFunc(func(row, column int) {
		log.Println("row", row, "column", column, "was selected")
		log.Println(table.GetCell(row, 0).GetReference())
		history := history(table.GetCell(row, 0).GetReference().(string), uptime.Hour)
		pager.AddPage("history", history, true, true)
	})
	return table
}

// updatetable refresh table with latest data.  do not call from main, only from goroutine
func updateTable(table *tview.Table, data []uptime.Status) {
	for i, row := range data {
		log.Println("updating table row", i)
		table.SetCell(i+2, 0, tview.NewTableCell(row.Site).
			SetAlign(tview.AlignCenter).SetExpansion(1).SetReference(row.Site))
		table.SetCell(i+2, 1, tview.NewTableCell(row.Status).
			SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(i+2, 2, tview.NewTableCell(fmt.Sprintf("%d", row.StatusCode)).
			SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(i+2, 3, tview.NewTableCell(row.Time.Format(time.RFC822)).
			SetAlign(tview.AlignCenter).SetExpansion(1))
	}
	app.Draw()
}

func monitorForm(dialog string) tview.Primitive {
	monitor := uptime.Monitor{}
	form := tview.NewForm().
		AddInputField("Name:", "", 0, nil, nil).
		AddInputField("Url:", "", 0, nil, nil).
		AddDropDown("Type", []string{"website", "ping"}, 0, nil).
		AddDropDown("Freq", []string{"1m", "5m", "30m", "60m"}, 0, nil).
		AddDropDown("Timeout", []string{"1s", "2s", "5s", "10s"}, 0, nil).
		AddButton("Cancel", func() {
			pager.RemovePage(dialog)
		})
	form.AddButton("Create", func() {
		monitor.Name = form.GetFormItem(0).(*tview.InputField).GetText()
		monitor.Url = form.GetFormItem(1).(*tview.InputField).GetText()
		selected, _ := form.GetFormItem(2).(*tview.DropDown).GetCurrentOption()
		monitor.Type = uptime.Type(selected)
		_, monitor.Freq = form.GetFormItem(3).(*tview.DropDown).GetCurrentOption()
		_, monitor.Timeout = form.GetFormItem(4).(*tview.DropDown).GetCurrentOption()
		db, err := uptime.OpenDB()
		if err != nil {
			log.Println("open database", err)
		}
		defer db.Close()
		if err := uptime.SaveMonitor(db, monitor); err != nil {
			log.Println("new monitor", err)
			pager.AddPage("error", errorDialog(err.Error()), true, true)
		}
		signalDaemon()
		pager.RemovePage(dialog)
	})
	form.SetBorder(true).SetTitle("Add Monitor").SetTitleAlign(tview.AlignCenter)
	return form
}

//func showLogs() tview.Primitive {
//logfile :=
//}

func validateURL(text string, last rune) bool {
	log.Println("validateURL", text)
	if !strings.Contains(text, "http://") {
		return false
	}
	return true
}

func signalDaemon() {
	bytes, err := os.ReadFile(filepath.Join(os.TempDir(), "uptimed.pid"))
	if err != nil {
		log.Println("read pid", err)
		return
	}
	pid, _ := strconv.Atoi(string(bytes))
	proc, err := os.FindProcess(pid)
	if err != nil {
		log.Println("no uptimed process", err)
		return
	}
	if err := proc.Signal(syscall.SIGHUP); err != nil {
		log.Println("signal uptimed", err)
	}
}
