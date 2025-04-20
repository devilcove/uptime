package main

import (
	"log"

	"github.com/devilcove/uptime"
	"github.com/rivo/tview"
)

func monitor(table *tview.Table) {
	log.Println("updating table")
	config := uptime.GetConfig()
	if config == nil {
		log.Println("no configuration ... bailing")
		return
	}
	db, err := uptime.OpenDB()
	defer db.Close()
	if err != nil {
		log.Println("database access", err)
		return
	}
	status, err := uptime.GetKeys(db, []string{"status"})
	if err != nil {
		log.Println("get status update", err)
		return
	}
	//app.QueueUpdateDraw(func() {
	updateTable(table, status)
	//})

}
