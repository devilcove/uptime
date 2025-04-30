package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logFile, err := os.Create("uptimed.log")
	if err != nil {
		log.Println("create log file", err)
	}
	log.SetOutput(logFile)
	log.Println("logfile opened")
	pid := os.Getpid()
	if err := os.WriteFile(filepath.Join(os.TempDir(), "uptimed.pid"), []byte(strconv.Itoa(pid)), fs.ModePerm); err != nil {
		log.Println("write pid file", err)
	}

	wg := sync.WaitGroup{}
	quit := make(chan os.Signal, 1)
	reset := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
	signal.Notify(reset, syscall.SIGHUP)
	ctx, cancel := context.WithCancel(context.Background())
	startMonitors(ctx, &wg)
	for {
		select {
		case <-quit:
			log.Println("quitting ...")
			cancel()
			wg.Wait()
			os.Remove(filepath.Join(os.TempDir(), "uptimed.pid"))
			return
		case <-reset:
			log.Println("restart monitors")
			cancel()
			wg.Wait()
			ctx, cancel = context.WithCancel(context.Background())
			startMonitors(ctx, &wg)
		}
	}
}

func startMonitors(ctx context.Context, wg *sync.WaitGroup) {
	monitorers, err := getMonitors()
	if err != nil {
		log.Println("get monitors", err)
	} else {
		for _, m := range monitorers {
			log.Println("starting monitor", m.Name)
			wg.Add(1)
			go monitor(ctx, wg, &m)
		}
	}
}
