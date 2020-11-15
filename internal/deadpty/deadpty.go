package deadpty

import "extraterm-wsl-proxy/internal/protocol"

type DeadPty struct {
}

func NewDeadPty(ptyID int, ptyActivity chan<- interface{}, errorMessage string) *DeadPty {
	this := new(DeadPty)

	go func() {
		outputMessage := protocol.OutputMessage{
			Message: protocol.Message{MessageType: "output"},
			Id:      ptyID,
			Data:    errorMessage,
		}
		ptyActivity <- outputMessage

		closedMessage := protocol.ClosedMessage{
			Message:  protocol.Message{MessageType: "closed"},
			Id:       ptyID,
			ExitCode: 0,
		}

		ptyActivity <- closedMessage
	}()

	return this
}

func (this *DeadPty) PermitDataSize(size int) {
}

func (this *DeadPty) Terminate() {
}

func (this *DeadPty) Write(data string) {
}

func (this *DeadPty) Resize(rows, columns int) {
}

func (this *DeadPty) GetWorkingDirectory() string {
	return ""
}
