package types

type Entity struct {
	ID         string
	Name       string
	Label      string
	Embedding  []float64
	Properties map[string]any
}

type Relation struct {
	SourceName string
	TargetName string
	Type       string
}