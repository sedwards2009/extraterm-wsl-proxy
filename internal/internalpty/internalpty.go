package internalpty

type InternalPty interface {
	GetChunk() []byte
	Terminate()
	Write(data string)
	Resize(rows, cols int)
}
