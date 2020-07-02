package realpty

import (
	"extraterm-go-proxy/internal/protocol"
	"extraterm-go-proxy/internal/utf8sanitize"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

type RealPty struct {
	pty     *os.File
	ptyLock sync.Mutex

	cmd *exec.Cmd

	permittedReadCond *sync.Cond
	permittedRead     int
}

const chunkSizeBytes = 10 * 1024
const chunkChannelSize = 10

func NewRealPty(cmd *exec.Cmd, ptyID int, ptyActivity chan<- interface{}, pty *os.File) *RealPty {
	this := new(RealPty)
	this.pty = pty
	this.cmd = cmd

	m := sync.Mutex{}
	this.permittedReadCond = sync.NewCond(&m)
	this.permittedRead = 0

	go this.readRoutine(ptyID, ptyActivity)

	return this
}

func (this *RealPty) readRoutine(ptyID int, ptyActivity chan<- interface{}) {
	sanitizer := utf8sanitize.NewUtf8Sanitizer()
	for {
		readSize := this.getPermittedSize()

		buffer := make([]byte, min(readSize, chunkSizeBytes))
		bufferSlice := buffer[:]
		n, err := this.pty.Read(bufferSlice)
		if err == nil {
			cleanData := sanitizer.Sanitize(bufferSlice[:n])
			if len(cleanData) != 0 {
				outputMessage := protocol.OutputMessage{
					Message: protocol.Message{MessageType: "output"},
					Id:      ptyID,
					Data:    string(bufferSlice[:n]),
				}

				this.decreasePermittedSize(n)
				ptyActivity <- outputMessage
			}
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

func (this *RealPty) getPermittedSize() int {
	this.permittedReadCond.L.Lock()
	if this.permittedRead <= 0 {
		this.permittedReadCond.Wait()
	}
	readSize := this.permittedRead
	this.permittedReadCond.L.Unlock()
	return readSize
}

func (this *RealPty) decreasePermittedSize(delta int) {
	this.permittedReadCond.L.Lock()
	this.permittedRead -= delta
	this.permittedReadCond.L.Unlock()
}

func (this *RealPty) PermitDataSize(size int) {
	this.permittedReadCond.L.Lock()
	this.permittedRead = size
	if this.permittedRead > 0 {
		this.permittedReadCond.Signal()
	}
	this.permittedReadCond.L.Unlock()
}

func (this *RealPty) Terminate() {
	this.cmd.Process.Kill()
	this.cmd.Wait()
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

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
