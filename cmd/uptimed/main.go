package main

import (
	"log"
	"os"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logFile, err := os.Create("uptimed.log")
	if err != nil {
		log.Println("create log file", err)
	}
	log.SetOutput(logFile)
	log.Println("logfile opened")
	monitorers, err := getMonitors()
	if err != nil {
		log.Println("get monitors", err)
	} else {
		for _, m := range monitorers {
			log.Println("starting monitor", m.Name)
			go monitor(&m)
		}
	}
	select {}
}
