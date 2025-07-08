// package to monitor uptime of servers
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
	// setup logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logFile, err := os.OpenFile("uptime.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err == nil {
		w := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(w)
	}
	// open database
	if err := openDB(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// signals, waitgroups and contexts
	wgMonitors := &sync.WaitGroup{}
	wgWeb := &sync.WaitGroup{}
	quit := make(chan os.Signal, 1)
	reset = make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
	signal.Notify(reset, syscall.SIGHUP)
	ctxMonitors, cancelMonitors := context.WithCancel(context.Background())
	ctxWeb, cancelWeb := context.WithCancel(context.Background())
	// start goroutines
	startMonitors(ctxMonitors, wgMonitors)
	wgWeb.Add(1)
	go web(ctxWeb, wgWeb)
	// wait for signals
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
			ctx, cancel := context.WithCancel(context.Background())
			cancelMonitors = cancel
			startMonitors(ctx, wgMonitors)
		}
	}
}
