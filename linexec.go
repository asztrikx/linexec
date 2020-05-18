package linexec

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

//Terminal which will get the commands in stdin
var Terminal = "bash"

//OutputLocation is the default outputlocation if it is ""
//outputLocation should be "" for no logs
//outputLocation should be "null" for no display
var OutputLocation string

var buffer []string

//Prepare collects commands to be executed together
func Prepare(cmd interface{}) {
	if commandS, ok := cmd.([]string); ok {
		buffer = append(buffer, commandS...)
	} else if command, ok := cmd.(string); ok {
		buffer = append(buffer, command)
	} else {
		panic("unknown type")
	}
}

//Finish executes collected commands by Prepare
//outputLocation should be "" for no logs
//outputLocation should be "null" for no display
func Finish(outputLocation string) string {
	s := Exec(buffer, outputLocation)
	buffer = []string{}
	return s
}

//Exec executes commands together
//buffer should be a string or []string
//outputLocation should be "" for no logs
//outputLocation should be "null" for no display
func Exec(buffer interface{}, outputLocation string) string {
	//buffer convert
	var commandS []string
	if a, ok := buffer.([]string); ok {
		commandS = a
	} else if a, ok := buffer.(string); ok {
		commandS = []string{a}
	} else {
		panic("unknown type")
	}

	//outputLocation format
	if outputLocation == "" && OutputLocation != "" {
		outputLocation = OutputLocation
	}
	if outputLocation != "" && outputLocation != "null" {
		outputFolder := outputLocation[:strings.LastIndex(outputLocation, "/")]
		Exec(fmt.Sprintf("mkdir -p %s", outputFolder), "null")
	}

	//Print header
	if outputLocation != "null" {
		fmt.Println()
		if outputLocation != "" {
			Exec(fmt.Sprintf(`printf "\n" >> %s`, outputLocation), "null")
		}

		timeText := fmt.Sprintf("## %s ##", time.Now().Format(time.RFC3339))
		delimiter := strings.Repeat("#", len(timeText))

		var header string
		header += delimiter + "\n"
		header += timeText + "\n"
		header += delimiter + "\n"
		for i := range commandS {
			header += fmt.Sprintf("## %s\n", commandS[i])
		}
		header += delimiter + "\n"

		fmt.Print(header)
		if outputLocation != "" {
			header = escape(header)
			Exec(fmt.Sprintf(`printf "%s" >> %s`, header, outputLocation), "null")
		}
	}

	//Prepare command
	var cmd *exec.Cmd
	cmd = exec.Command(terminal)
	var stdin io.WriteCloser
	var stdout io.ReadCloser
	var err error
	if stdin, err = cmd.StdinPipe(); err != nil {
		panic(err)
	}
	if stdout, err = cmd.StdoutPipe(); err != nil {
		panic(err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	//Output
	var output string
	stdoutChan := make(chan bool, 1)
	go Printer(&stdout, stdoutChan, outputLocation, &output)

	//Execute command
	command := strings.Join(commandS, "\n") + "\nexit\n"
	io.WriteString(stdin, command)
	stdin.Close()

	//Waiting for output print to finish
	<-stdoutChan
	stdout.Close()
	//[I] Save output in one chuck to avoid fork resource exhaustion
	if outputLocation != "" && outputLocation != "null" {
		Exec(fmt.Sprintf(`printf "%s" >> %s`, escape(output), outputLocation), "null")
	}

	//Format ending
	if outputLocation != "null" {
		fmt.Println()
		if outputLocation != "" {
			Exec(fmt.Sprintf(`printf "\n" >> %s`, outputLocation), "null")
		}
	}

	return output
}

//Printer echos to the correct location
func Printer(stdout *io.ReadCloser, stdoutChan chan bool, outputLocation string, output *string) {
	b := make([]byte, 100)
	for {
		n, err := (*stdout).Read(b)
		if err != nil {
			break
		}
		*output += string(b[:n])
		if outputLocation != "null" {
			fmt.Print(string(b[:n]))
		}
	}
	stdoutChan <- true
}

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}
