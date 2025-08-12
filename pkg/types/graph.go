package types

type Subgraph struct {
	Entities  []Entity
	Relations []Relation
}

type EvaluationResult struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}