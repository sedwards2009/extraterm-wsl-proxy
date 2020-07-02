package deadpty

import "extraterm-go-proxy/internal/protocol"

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
			Message: protocol.Message{MessageType: "closed"},
			Id:      ptyID,
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
