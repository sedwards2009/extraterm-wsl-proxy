package main

import (
	"bufio"
	"encoding/json"
	"extraterm-wsl-proxy/internal/deadpty"
	"extraterm-wsl-proxy/internal/envmaputils"
	"extraterm-wsl-proxy/internal/internalpty"
	"extraterm-wsl-proxy/internal/protocol"
	"extraterm-wsl-proxy/internal/realpty"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

const logFineFlag = true

type appState struct {
	idCounter   int
	ptyPairsMap map[int]internalpty.InternalPty
	ptyActivity chan interface{}
}

func main() {
	var appState appState
	appState.ptyPairsMap = map[int]internalpty.InternalPty{}

	commandChan := make(chan []byte, 1)
	appState.ptyActivity = make(chan interface{}, 1)

	go commandLoop(commandChan)

	for {
		select {

		case commandLine := <-commandChan:
			appState.processCommand(commandLine)

		case message := <-appState.ptyActivity:
			appState.processPtyActivity(message)
		}
	}
}

func commandLoop(output chan<- []byte) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		output <- line
	}
}

func (appState *appState) processCommand(commandLine []byte) {
	var msg interface{}
	json.Unmarshal(commandLine, &msg)

	switch msgMap := msg.(type) {
	case map[string]interface{}:
		commandType := msgMap["type"].(string)

		switch commandType {

		case "create":
			appState.handleCreate(commandLine)

		case "write":
			appState.handleWrite(commandLine)

		case "resize":
			appState.handleResize(commandLine)

		case "permit-data-size":
			appState.handlePermitDataSize(commandLine)

		case "close":
			appState.handleClose(commandLine)

		case "get-working-directory":
			appState.handleGetWorkingDirectory(commandLine)

		case "terminate":
			os.Exit(0)

		default:
			fmt.Printf("Unknown command command type '%s'.", commandType)
			os.Exit(1)
		}
	}
}

func sendToController(msg interface{}) {
	jsonString, _ := json.Marshal(msg)
	os.Stdout.Write(jsonString)
	os.Stdout.Write([]byte{'\n'})
}

func (appState *appState) handleCreate(line []byte) {
	var msg protocol.CreateMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}

	var envMap *map[string]string
	if msg.Env != nil {
		envMap = msg.Env
	} else {
		envMap = envmaputils.KeyValueArrayToMap(os.Environ())
	}

	// Add in the extra value from msg.extraEnv
	if msg.ExtraEnv != nil {
		for key, value := range *msg.ExtraEnv {
			(*envMap)[key] = value
		}
	}
	env := envmaputils.KeyValueMapToArray(envMap)
	// Set up the default working directory
	var cwd = ""
	if msg.SuggestedCwd != nil && *msg.SuggestedCwd != "" {
		if _, err := os.Stat(*msg.SuggestedCwd); err == nil {
			cwd = *msg.SuggestedCwd
		}
	}

	if cwd == "" && msg.Cwd != nil && *msg.Cwd != "" {
		cwd = *msg.Cwd
	}

	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	if _, err := os.Stat(cwd); err != nil {
		if os.IsNotExist(err) {
			cwd, _ = os.Getwd()
		} else {
			log.Fatal("Received unexpected error while checking cwd. ", err)
		}
	}

	cmd := exec.Command(msg.Argv[0])
	cmd.Env = *env
	cmd.Args = msg.Argv[1:]
	cmd.Dir = cwd

	appState.idCounter++
	ptyID := appState.idCounter

	processParameters := formatNewProcessParameters(msg.Argv, cwd, *env)
	log.Print(fmt.Sprintf("Starting WSL process: %s\n", processParameters))

	var newPty internalpty.InternalPty
	var winsize = pty.Winsize{Rows: uint16(msg.Columns), Cols: uint16(msg.Rows), X: 8, Y: 8}
	pty, err := pty.StartWithSize(cmd, &winsize)
	if err != nil {
		errorMessage := fmt.Sprintf("Error while starting process '%s'. %s\n%s", msg.Argv[0], err, processParameters)
		log.Print(errorMessage)
		newPty = deadpty.NewDeadPty(ptyID, appState.ptyActivity, errorMessage)
	} else {
		newPty = realpty.NewRealPty(cmd, ptyID, appState.ptyActivity, pty)
	}

	appState.ptyPairsMap[ptyID] = newPty
	sendToController(protocol.CreatedMessage{Message: protocol.Message{MessageType: "created"}, Id: ptyID})
}

func formatNewProcessParameters(argv []string, cwd string, env []string) string {
	return fmt.Sprintf("argv: %s, cwd: '%s', env: %s", formatStringArray(argv), cwd, formatStringArray(env))
}

func formatStringArray(strArray []string) string {
	result := "["
	comma := ""
	for _, item := range strArray {
		result = fmt.Sprintf("%s%s'%s'", result, comma, item)
		comma = ", "
	}
	result = result + "]"
	return result
}

func (appState *appState) handleWrite(line []byte) {
	var msg protocol.WriteMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}

	if pty, ok := (*appState).ptyPairsMap[msg.Id]; ok {
		pty.Write(msg.Data)
	}
}

func (appState *appState) handleResize(line []byte) {
	var msg protocol.ResizeMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}

	if pty, ok := (*appState).ptyPairsMap[msg.Id]; ok {
		pty.Resize(msg.Rows, msg.Columns)
	}
}

func (appState *appState) handlePermitDataSize(line []byte) {
	var msg protocol.PermitDataSizeMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}

	if pty, ok := (*appState).ptyPairsMap[msg.Id]; ok {
		pty.PermitDataSize(msg.Size)
	}
}

func (appState *appState) handleClose(line []byte) {
	var msg protocol.CloseMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}

	if pty, ok := (*appState).ptyPairsMap[msg.Id]; ok {
		pty.Terminate()
	}
}

func (appState *appState) handleGetWorkingDirectory(line []byte) {
	var msg protocol.GetWorkingDirectoryMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}

	var cwd string = ""
	if pty, ok := (*appState).ptyPairsMap[msg.Id]; ok {
		cwd = pty.GetWorkingDirectory()
	}

	reply := protocol.GetWorkingDirectoryMessage{
		Message: protocol.Message{MessageType: "working-directory"},
		Id:      msg.Id,
		Cwd:     cwd,
	}

	sendToController(reply)
}

func (appState *appState) processPtyActivity(message interface{}) {
	switch closedMessage := message.(type) {
	case protocol.ClosedMessage:
		delete(appState.ptyPairsMap, closedMessage.Id)
	default:
	}
	sendToController(message)
}

func logFine(format string, args ...interface{}) {
	if logFineFlag {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
