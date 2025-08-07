package types

type Entity struct {
	ID         string
	Name       string
	Label      string
	Embedding  []float32
	Properties map[string]any
}

type Relation struct {
	SourceName string
	TargetName string
	Type       string
}

type ParsedData struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}