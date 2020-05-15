package realpty

import (
	"os"

	"github.com/creack/pty"
)

type RealPty struct {
	stdin       chan []byte // Subprocess stdin
	stdoutChan  chan []byte
	pty         *os.File
	ptyActivity chan bool
}

const chunkSizeBytes = 10 * 1024
const chunkChannelSize = 10

func NewRealPty(pty *os.File, ptyActivity chan bool) *RealPty {
	this := new(RealPty)
	this.pty = pty
	this.ptyActivity = ptyActivity

	stdoutChan := make(chan []byte, 1)
	this.stdoutChan = stdoutChan

	go this.readRoutine()

	return this
}

func (this *RealPty) readRoutine() {
	for {
		buffer := make([]byte, chunkSizeBytes)
		bufferSlice := buffer[:]
		n, err := this.pty.Read(bufferSlice)
		if err != nil {
			// logFine("pth.Read() errored %s", err)

			break
		}
		this.stdoutChan <- bufferSlice[:n]
		this.ptyActivity <- true
	}
}

func (this *RealPty) GetChunk() []byte {
	select {
	case chunk := <-this.stdoutChan:
		return chunk
	default:
		return nil
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
