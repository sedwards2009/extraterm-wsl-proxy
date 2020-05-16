package internalpty

type InternalPty interface {
	Terminate()
	Write(data string)
	Resize(rows, cols int)
}
