package utils

import (
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"strings"
)

func SubgraphToString(subgraph *types.Subgraph) string {
	if subgraph == nil || len(subgraph.Entities) == 0 {
		return "No relevant information found in the knowledge graph."
	}

	var sb strings.Builder
	sb.WriteString("Found Entities and their relationships:\n")

	for _, entity := range subgraph.Entities {
		sb.WriteString(fmt.Sprintf("\n- Entity: %s (Type: %s)\n", entity.Name, entity.Label))
		for _, relation := range subgraph.Relations {
			if relation.SourceName == entity.Name {
				sb.WriteString(fmt.Sprintf("  - [%s] --(%s)--> [%s]\n", relation.SourceName, relation.Type, relation.TargetName))
			}
			if relation.TargetName == entity.Name {
			}
		}
	}
	return sb.String()
}