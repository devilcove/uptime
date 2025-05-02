package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var reset chan os.Signal

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logFile, err := os.OpenFile("uptime.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err == nil {
		w := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(w)
	}
	if err := openDB(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	wgMonitors := &sync.WaitGroup{}
	wgWeb := &sync.WaitGroup{}
	quit := make(chan os.Signal, 1)
	reset = make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
	signal.Notify(quit, syscall.SIGHUP)
	ctxMonitors, cancelMonitors := context.WithCancel(context.Background())
	startMonitors(ctxMonitors, wgMonitors)
	ctxWeb, cancelWeb := context.WithCancel(context.Background())
	wgWeb.Add(1)
	go web(ctxWeb, wgWeb)
	for {
		select {
		case <-quit:
			log.Println("quitting ...")
			cancelMonitors()
			cancelWeb()
			wgMonitors.Wait()
			wgWeb.Wait()
			return
		case <-reset:
			log.Println("reset monitors")
			cancelMonitors()
			wgMonitors.Wait()
			ctxMonitors, cancelMonitors = context.WithCancel(context.Background())
			startMonitors(ctxMonitors, wgMonitors)
		}
	}
}
