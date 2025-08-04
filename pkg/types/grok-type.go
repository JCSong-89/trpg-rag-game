package types

type GrokMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GrokChoice struct {
	Message GrokMessage `json:"message"`
}