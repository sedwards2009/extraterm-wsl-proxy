package realpty

import (
	"C"
	"extraterm-wsl-proxy/internal/protocol"
	"extraterm-wsl-proxy/internal/utf8sanitize"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

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
			this.cmd.Wait()
			exitCode := this.cmd.ProcessState.ExitCode()

			closedMessage := protocol.ClosedMessage{
				Message:  protocol.Message{MessageType: "closed"},
				Id:       ptyID,
				ExitCode: exitCode,
			}
			ptyActivity <- closedMessage

			this.ptyLock.Lock()
			this.pty.Close()
			this.pty = nil
			this.ptyLock.Unlock()

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

func (this *RealPty) GetWorkingDirectory() string {
	this.ptyLock.Lock()
	defer this.ptyLock.Unlock()
	if this.pty == nil {
		return ""
	}

	var pid C.uint
	if err := ioctl(this.pty.Fd(), syscall.TIOCGPGRP, uintptr(unsafe.Pointer(&pid))); err != nil {
		return ""
	}

	if dir, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil {
		return dir
	}
	return ""
}

func ioctl(fd, cmd, ptr uintptr) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
	if e != 0 {
		return e
	}
	return nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
