package main

import (
	"log"
	"os"
	"io"
)

func main() {
	// Set up logging
	log.SetOutput(os.Stderr)
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags)
	logFile, err := os.OpenFile("log.txt",
		os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Couldn't open log file.")
	}
	defer func() {
		err = logFile.Close()
		if err != nil {
			log.Print("Log file was not closed properly. ", err)
		}
	}()
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	matchSequencesCaller()
}