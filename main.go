package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var sigdone chan bool

var majversion string
var minversion string
var builddate string

func main() {
	log.Println("Go Simple Web Server")
	log.Printf("Version: %s.%s", majversion, minversion)
	log.Printf("Built: %s", builddate)
	log.Println("Http server starting")
	sigdone = make(chan bool, 1)
	cfg := loadConfig()
	runtime.GOMAXPROCS(runtime.NumCPU() * 4)
	srv := newServer(cfg)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	running := true
	signaled := false
	for {
		select {
		case <-srv.Stopped:
			log.Println("Http server stopped")
			if !running {
				return
			}
			srv.Start()
		case <-sigdone:
			running = false
			srv.Stop()
		case s := <-sig:
			log.Println("Got signal:", s)
			if s == syscall.SIGHUP {
				newCfg := loadConfig()
				*cfg = *newCfg
				srv.Stop()
				break
			}
			if signaled {
				log.Println("Force exit")
				os.Exit(137)
				return
			}
			signaled = true
			sigdone <- true
		}
	}
}
