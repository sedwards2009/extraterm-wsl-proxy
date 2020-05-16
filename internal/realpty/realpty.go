package realpty

import (
	"extraterm-go-proxy/internal/protocol"
	"os"

	"github.com/creack/pty"
)

type RealPty struct {
	pty *os.File
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

			ptyActivity <- closedMessage
			close(this.pty)
			this.pty = nil
			break
		}
	}
}

func (this *RealPty) Terminate() {
	this.pty.Close()
	this.pty = nil
}

func (this *RealPty) Write(data string) {
	this.pty.Write([]byte(data))
}

func (this *RealPty) Resize(rows, columns int) {
	winsize := pty.Winsize{Rows: uint16(rows), Cols: uint16(columns), X: 8, Y: 8}
	pty.Setsize(this.pty, &winsize)
}
