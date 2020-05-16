package realpty

import (
	"extraterm-go-proxy/internal/protocol"
	"os"
	"sync"
	"github.com/creack/pty"
)

type RealPty struct {
	pty *os.File
	ptyLock sync.Mutex
}

const chunkSizeBytes = 10 * 1024
const chunkChannelSize = 10

func NewRealPty(ptyID int, ptyActivity chan<- interface{}, pty *os.File) *RealPty {
	this := new(RealPty)
	this.pty = pty

	go this.readRoutine(ptyID, ptyActivity)

	return this
}

func (this *RealPty) readRoutine(ptyID int, ptyActivity chan<- interface{}) {
	for {
		buffer := make([]byte, chunkSizeBytes)
		bufferSlice := buffer[:]
		n, err := this.pty.Read(bufferSlice)
		if err == nil {
			outputMessage := protocol.OutputMessage{
				Message: protocol.Message{MessageType: "output"},
				Id:      ptyID,
				Data:    string(bufferSlice[:n]),
			}
			ptyActivity <- outputMessage
		} else {
			closedMessage := protocol.ClosedMessage{
				Message: protocol.Message{MessageType: "closed"},
				Id:      ptyID,
			}

			this.ptyLock.Lock()
			this.pty.Close()
			this.pty = nil
			this.ptyLock.Unlock()

			ptyActivity <- closedMessage
			break
		}
	}
}

func (this *RealPty) Terminate() {
	this.ptyLock.Lock()
	defer this.ptyLock.Unlock()
	if this.pty == nil {
		return
	}

	this.pty.Close()
}

func (this *RealPty) Write(data string) {
	this.ptyLock.Lock()
	defer this.ptyLock.Unlock()
	if this.pty == nil {
		return
	}

	this.pty.Write([]byte(data))
}

func (this *RealPty) Resize(rows, columns int) {
	this.ptyLock.Lock()
	defer this.ptyLock.Unlock()
	if this.pty == nil {
		return
	}

	winsize := pty.Winsize{Rows: uint16(rows), Cols: uint16(columns), X: 8, Y: 8}
	pty.Setsize(this.pty, &winsize)
}