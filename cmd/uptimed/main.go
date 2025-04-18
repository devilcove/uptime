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
	monitorers := getMonitors()
	for _, m := range monitorers {
		go monitor(&m)
	}
	select {}
}
