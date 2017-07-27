package main

import (
	"errors"
	"log"
	"os/exec"
	"bytes"
	"time"
)

func newErr(input string, err error) error {
	return errors.New(input + " " + err.Error())
}

// Outputs a system command to log with all output on error.
func commandVerboseOnErr(input string) (string, string, error) {
	stdout, stderr, err := commandWithOutput(input)
	if err != nil {
		log.Print("Command: " + input)
		if stdout != "" {
			log.Print(stdout)
		}
		if stderr != "" {
			log.Print(stderr)
		}
		err = newErr("Error in running command.", err)
		log.Print(err)
	}
	return stdout, stderr, err
}

// Outputs a system command to log with stdout, stderr, and err output.
func commandVerbose(input string) (string, string, error) {
	log.Print("Command: " + input)
	stdout, stderr, err := commandWithOutput(input)
	if stdout != "" {
		log.Print(stdout)
	}
	if stderr != "" {
		log.Print(stderr)
	}
	if err != nil {
		err = newErr("Error in running command.", err)
		log.Print(err)
	} else {
		log.Print("Command ran with no errors.")
	}
	return stdout, stderr, err
}

// Executes a shell command and returns the stdout, stderr, and err
func commandWithOutput(input string) (string, string, error) {
	cmd := exec.Command("sh", "-cx", input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outResp := stdout.String()
	errResp := stderr.String()
	return outResp, errResp, err
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}