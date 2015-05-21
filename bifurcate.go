package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"text/template"
	"time"
)

type Configuration struct {
	Programs map[string]ConfigurationProgram
}

type ConfigurationProgram struct {
	Args     []string
	Requires []ConfigurationProgramRequires
}

type ConfigurationProgramRequires struct {
	File string
}

func waitForRequires(name string, requires []ConfigurationProgramRequires) {
	success := false
	for !success {
		success = true
		for _, req := range requires {
			if req.File != "" {
				filename := req.File
				if _, err := os.Stat(filename); os.IsNotExist(err) {
					fmt.Println(filename, "Not found")
					success = false
				} else {
					fmt.Println("Found", filename)
				}
			}
		}

		if !success {
			fmt.Println("Still waiting on", name, "requires")
			time.Sleep(1 * time.Second)
		}
	}
}

func waitFor(name string, requires []ConfigurationProgramRequires, cmd *exec.Cmd, quit chan int) {
	if len(requires) > 0 {
		fmt.Println("Making sure that all requirements are good for", name)
		waitForRequires(name, requires)
	}

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
			fmt.Println("Error command error was not 'exit err' for", name, err)
			quit <- 1
		}
	} else {
		quit <- 0
	}
}

func readConfiguration(configFilePath string) Configuration {
	// read config file in as string
	configFileBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		fmt.Println("Unable to read config", configFilePath, err)
		os.Exit(1)
	}

	// convert template file into json
	var doc bytes.Buffer
	funcMap := template.FuncMap{
		"env": os.Getenv,
	}
	t, templateError := template.New("config").Funcs(funcMap).Parse(string(configFileBytes))
	if templateError != nil {
		fmt.Println("Error reading template", configFilePath, templateError)
		os.Exit(1)
	}
	t.Execute(&doc, nil)

	// read json into configuration object
	configuration := Configuration{}
	err = json.Unmarshal([]byte(doc.String()), &configuration)
	if err != nil {
		fmt.Println("error parsing config", configFilePath, err)
		os.Exit(1)
	}
	return configuration
}

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Println("Please pass the configuration file to use as the first argument")
		os.Exit(1)
	}
	configFilePath := args[1]
	configuration := readConfiguration(configFilePath)

	// start catching signals early
	c := make(chan os.Signal, 1)
	signal.Notify(c)

	routineQuit := make(chan int)
	var cmds []*exec.Cmd

	// check validity
	for name, program := range configuration.Programs {
		if len(program.Args) <= 0 {
			fmt.Println("Unable to run empty command", name)
			os.Exit(1)
		}
	}

	if os.Getppid() != 0 {
		fmt.Println("I am PID", os.Getpid(), "it would be better if I was PID 1")
	}
	// run them
	for name, program := range configuration.Programs {
		cmdArgs := program.Args
		filePath := cmdArgs[0]
		cmd := exec.Command(filePath, cmdArgs[1:]...)
		cmds = append(cmds, cmd)
		go waitFor(name, program.Requires, cmd, routineQuit)
	}

	// catch any and all signals and forward them to all child commands
	go func() {
		s := <-c
		fmt.Println("Got signal", s, "sending along")
		for _, cmd := range cmds {
			if cmd.Process != nil {
				cmd.Process.Signal(s)
			}
		}
	}()

	// if anything dies kill the rest and then exit ourselves
	ret := <-routineQuit
	fmt.Println("Killing everything and shutting down")
	for _, cmd := range cmds {
		if cmd.Process != nil {
			cmd.Process.Signal(os.Kill)
		}
	}
	os.Exit(ret)
}
