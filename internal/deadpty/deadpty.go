package deadpty

type DeadPty struct {
	sentMsg bool
	message string
}

func NewDeadPty(message string, ptyActivity chan bool) *DeadPty {
	this := new(DeadPty)
	this.message = message

	go func() {
		ptyActivity <- true
	}()

	return this
}

func (this *DeadPty) GetChunk() []byte {
	if !this.sentMsg {
		this.sentMsg = true
		return []byte(this.message)
	}
	return nil
}

func (this *DeadPty) Terminate() {
}

func (this *DeadPty) Write(data string) {
}

func (this *DeadPty) Resize(rows, columns int) {
}
