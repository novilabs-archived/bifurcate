package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

type Configuration struct {
	Programs map[string][]string
}

func waitFor(name string, cmd *exec.Cmd, quit chan int) {
	fmt.Println("Running", name, "cmd:", cmd.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitStatus := status.ExitStatus()
				fmt.Println("Saw exit status of", exitStatus, "for", name)
				quit <- exitStatus
			} else {
				fmt.Println("Unable to get exit status of", name)
				quit <- 1
			}
		} else {
			fmt.Println("Error not exit err for", name)
			quit <- 1
		}
	} else {
		quit <- 0
	}
}

func main() {

	args := os.Args
	if len(args) != 2 {
		fmt.Println("Please pass the configuration file to use as the first argument")
		os.Exit(1)
	}
	configFilePath := args[1]
	file, _ := os.Open(configFilePath)
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error parsing config", configFilePath, err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c)

	routineQuit := make(chan int)
	var cmds []*exec.Cmd

	// check validity
	for name, cmdArgs := range configuration.Programs {
		if len(cmdArgs) <= 0 {
			fmt.Println("Unable to run empty command", name)
			os.Exit(1)
		}
	}

	// run them
	for name, cmdArgs := range configuration.Programs {
		filePath := cmdArgs[0]
		cmd := exec.Command(filePath, cmdArgs[1:]...)
		cmds = append(cmds, cmd)
		go waitFor(name, cmd, routineQuit)
	}

	go func() {
		s := <-c
		fmt.Println("Got signal", s, "sending along")
		for _, cmd := range cmds {
			cmd.Process.Signal(s)
		}
	}()

	ret := <-routineQuit
	fmt.Println("Killing everything and shutting down")

	// if anything dies kill the rest and then exit ourselves
	for _, cmd := range cmds {
		cmd.Process.Signal(os.Kill)
	}

	os.Exit(ret)
}
