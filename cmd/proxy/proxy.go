package main

import (
	"bufio"
	"encoding/json"
	"extraterm-go-proxy/internal/envmaputils"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

const logFineFlag = true

//-------------------------------------------------------------------------
type internalPty interface {
	getChunk() []byte
	terminate()
	write(data string)
	resize(rows, cols int)
}

type Message struct {
	MessageType string `json:"type"`
}

type createMessage struct {
	Message
	Argv     []string           `json:"argv"`
	Cwd      *string            `json:"cwd"`
	Rows     float64            `json:"rows"`
	Columns  float64            `json:"columns"`
	Env      *map[string]string `json:"env"`
	ExtraEnv *map[string]string `json:"extraEnv"`
}

type createdMessage struct {
	Message
	Id int `json:"id"`
}

type writeMessage struct {
	Message
	Id   int    `json:"id"`
	Data string `json:"data"`
}

type resizeMessage struct {
	Message
	Id      int `json:"id"`
	Rows    int `json:"rows"`
	Columns int `json:"columns"`
}

type permitDataSizeMessage struct {
	Message
	Id   int `json:"id"`
	Size int `json:"size"`
}

type closeMessage struct {
	Message
	Id int `json:"id"`
}

type outputMessage struct {
	Message
	Id   int    `json:"id"`
	Data string `json:"data"`
}

type appState struct {
	idCounter   int
	ptyPairsMap map[int]internalPty
	ptyActivity chan bool
}

//-------------------------------------------------------------------------
func main() {
	var appState appState
	appState.ptyPairsMap = map[int]internalPty{}

	commandChan := make(chan []byte, 1)
	appState.ptyActivity = make(chan bool, 1)

	go commandLoop(commandChan)

	for {
		select {
		case commandLine := <-commandChan:
			logFine("main thread. Read: %s", commandLine)

			appState.processCommand(commandLine)
		case <-appState.ptyActivity:
			appState.checkPtyOutput()
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
	logFine("sendToController(%s)", jsonString)
	os.Stdout.Write(jsonString)
	os.Stdout.Write([]byte{'\n'})
}

func (appState *appState) handleCreate(line []byte) {
	var msg createMessage
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
	var cwd *string = msg.Cwd
	if cwd == nil || *cwd == "" {
		cwd = nil
	} else {
		if _, err := os.Stat(*cwd); err != nil {
			if os.IsNotExist(err) {
				cwd = nil
			} else {
				log.Fatal(err)
			}
		}
	}

	cmd := exec.Command(msg.Argv[0])
	cmd.Env = *env
	cmd.Args = msg.Argv[1:]
	// cmd.Dir = *cwd	// TODO

	var newPty internalPty
	var winsize = pty.Winsize{Rows: uint16(msg.Columns), Cols: uint16(msg.Rows), X: 8, Y: 8}
	pty, err := pty.StartWithSize(cmd, &winsize)
	if err != nil {
		message := fmt.Sprintf("Error while starting process '%s'. %s", msg.Argv[0], err)
		log.Print(message)
		newPty = newDeadPty(message, appState.ptyActivity)
	} else {
		newPty = newRealPty(pty, appState.ptyActivity)
	}

	appState.idCounter++
	appState.ptyPairsMap[appState.idCounter] = newPty
	sendToController(createdMessage{Message: Message{"created"}, Id: appState.idCounter})
}

func (appState *appState) handleWrite(line []byte) {
	var msg writeMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}
	// TODO handle unknown ID
	(*appState).ptyPairsMap[msg.Id].write(msg.Data)
}

func (appState *appState) handleResize(line []byte) {
	var msg resizeMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}

	(*appState).ptyPairsMap[msg.Id].resize(msg.Rows, msg.Columns)
}

func (appState *appState) handlePermitDataSize(line []byte) {
	var msg permitDataSizeMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}
	// TODO
	// (*appState).ptyPairsMap[msg.id].internalPty.permitDataSize(msg.size)
}

func (appState *appState) handleClose(line []byte) {
	var msg closeMessage
	err := json.Unmarshal(line, &msg)
	if err != nil {
		log.Fatal(err)
	}

	(*appState).ptyPairsMap[msg.Id].terminate()

}

func (appState *appState) checkPtyOutput() {
	for id, internalPty := range appState.ptyPairsMap {
		chunk := internalPty.getChunk()
		if chunk != nil {
			msg := outputMessage{Message{"output"}, id, string(chunk)}
			sendToController(msg)
		}
	}
}

//-------------------------------------------------------------------------
type realPty struct {
	stdin       chan []byte // Subprocess stdin
	stdoutChan  chan []byte
	pty         *os.File
	ptyActivity chan bool
}

const chunkSizeBytes = 10 * 1024
const chunkChannelSize = 10

func newRealPty(pty *os.File, ptyActivity chan bool) *realPty {
	this := new(realPty)
	this.pty = pty
	this.ptyActivity = ptyActivity

	stdoutChan := make(chan []byte, 1)
	this.stdoutChan = stdoutChan

	go this.readRoutine()

	return this
}

func (this *realPty) readRoutine() {
	for {
		buffer := make([]byte, chunkSizeBytes)
		bufferSlice := buffer[:]
		n, err := this.pty.Read(bufferSlice)
		if err != nil {
			logFine("pth.Read() errored %s", err)

			break
		}
		this.stdoutChan <- bufferSlice[:n]
		this.ptyActivity <- true
	}
}

func (this *realPty) getChunk() []byte {
	select {
	case chunk := <-this.stdoutChan:
		return chunk
	default:
		return nil
	}
}

func (this *realPty) terminate() {
	this.pty.Close()
	this.pty = nil
}

func (this *realPty) write(data string) {
	this.pty.Write([]byte(data))
}

func (this *realPty) resize(rows, columns int) {
	winsize := pty.Winsize{Rows: uint16(rows), Cols: uint16(columns), X: 8, Y: 8}
	pty.Setsize(this.pty, &winsize)
}

//-------------------------------------------------------------------------
type deadPty struct {
	sentMsg bool
	message string
}

func newDeadPty(message string, ptyActivity chan bool) *deadPty {
	this := new(deadPty)
	this.message = message

	go func() {
		ptyActivity <- true
	}()

	return this
}

func (this *deadPty) getChunk() []byte {
	if !this.sentMsg {
		this.sentMsg = true
		return []byte(this.message)
	}
	return nil
}

func (this *deadPty) terminate() {
}

func (this *deadPty) write(data string) {
}

func (this *deadPty) resize(rows, columns int) {
}

func logFine(format string, args ...interface{}) {
	if logFineFlag {
		fmt.Fprintf(os.Stderr, format, args)
	}
}
