package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func gracefulShutdown(sigs chan os.Signal, done chan bool) {
	signal.Notify(sigs, syscall.SIGTERM)
	sig := <-sigs
	switch sig.String() {
	case syscall.SIGTERM.String():
		log.Println("graceful shutdown...")
		// TODO: decide appropriate sleep second
		time.Sleep(5 * time.Second)
	}
	log.Printf("Get signal: %s\n", sig.String())
	done <- true
}
