package protocol

type Message struct {
	MessageType string `json:"type"`
}

type CreateMessage struct {
	Message
	Argv     []string           `json:"argv"`
	Cwd      *string            `json:"cwd"`
	Rows     float64            `json:"rows"`
	Columns  float64            `json:"columns"`
	Env      *map[string]string `json:"env"`
	ExtraEnv *map[string]string `json:"extraEnv"`
}

type CreatedMessage struct {
	Message
	Id int `json:"id"`
}

type WriteMessage struct {
	Message
	Id   int    `json:"id"`
	Data string `json:"data"`
}

type ResizeMessage struct {
	Message
	Id      int `json:"id"`
	Rows    int `json:"rows"`
	Columns int `json:"columns"`
}

type PermitDataSizeMessage struct {
	Message
	Id   int `json:"id"`
	Size int `json:"size"`
}

type CloseMessage struct {
	Message
	Id int `json:"id"`
}

type OutputMessage struct {
	Message
	Id   int    `json:"id"`
	Data string `json:"data"`
}
