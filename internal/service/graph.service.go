package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/internal/llm"
	"github.com/JCSong-89/trpg-rag-game/internal/prompt"
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"github.com/JCSong-89/trpg-rag-game/pkg/utils"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"log"
	"strings"
)

func EvaluateSubgraphWithLLM(ctx context.Context, subgraph *types.Subgraph, query string) (*types.EvaluationResult, error) {
	var sb strings.Builder

	sb.WriteString("Entities:\n")
	for _, e := range subgraph.Entities {
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", e.Name, e.Label))
	}

	sb.WriteString("\nRelations:\n")
	for _, r := range subgraph.Relations {
		sb.WriteString(fmt.Sprintf("- %s -> [%s] -> %s\n", r.SourceName, r.Type, r.TargetName))
	}

	subgraphText := sb.String()
	prompt := fmt.Sprintf(prompt.EvaluatePromptTemplate, query, subgraphText)

	responseText, err := llm.GenerateContentWithHTTP(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("Gemini 평가 API 호출 실패: %w", err)
	}

	jsonString, err := utils.ExtractJSONFromString(responseText)
	if err != nil {
		return nil, fmt.Errorf("응답에서 JSON 추출 실패: %w", err)
	}

	var result types.EvaluationResult
	if err := json.Unmarshal([]byte(jsonString), &result); err != nil {
		return nil, fmt.Errorf("JSON 응답 파싱 실패: %w, 원본 응답: %s", err, responseText)
	}

	return &result, nil
}

func FuseSubgraph(ctx context.Context, subgraphs []*types.Subgraph, query string) *types.Subgraph {
	var bestSubgraph *types.Subgraph
	maxScore := -1.0

	log.Println("LLM을 이용한 서브그래프 평가를 시작합니다...")

	for _, sg := range subgraphs {
		if sg == nil || len(sg.Entities) == 0 {
			continue
		}

		evalResult, err := EvaluateSubgraphWithLLM(ctx, sg, query)
		if err != nil {
			log.Printf("경고: 서브그래프 평가 중 오류 발생: %v", err)
			continue
		}

		log.Printf("평가 결과: 점수=%.2f, 이유=%s", evalResult.Score, evalResult.Reason)

		if evalResult.Score > maxScore {
			maxScore = evalResult.Score
			bestSubgraph = sg
		}
	}

	if bestSubgraph == nil {
		log.Println("경고: 유효한 서브그래프를 선택하지 못했습니다. 비어있는 서브그래프를 반환합니다.")
		return &types.Subgraph{}
	}

	log.Printf("최고 점수(%.2f)의 서브그래프를 선택했습니다.", maxScore)
	return bestSubgraph
}

func parseSubgraphFromRecords(records []*neo4j.Record) *types.Subgraph {
	subgraph := &types.Subgraph{}
	if len(records) == 0 {
		return subgraph
	}
	record := records[0]

	entitiesMap := make(map[string]types.Entity)

	if nodesInterface, ok := record.Get("nodes"); ok {
		nodes := nodesInterface.([]interface{})

		for _, nodeInterface := range nodes {
			node := nodeInterface.(neo4j.Node)

			if _, exists := entitiesMap[node.ElementId]; !exists {
				entitiesMap[node.ElementId] = types.Entity{
					ID:         node.ElementId,
					Name:       node.Props["name"].(string),
					Label:      node.Labels[0],
					Properties: node.Props,
				}
			}
		}
	}

	if relationsInterface, ok := record.Get("rels"); ok {
		relations := relationsInterface.([]interface{})
		for _, relInterface := range relations {
			rel := relInterface.(neo4j.Relationship)
			startNode := entitiesMap[rel.StartElementId]
			endNode := entitiesMap[rel.EndElementId]

			if startNode.Name != "" && endNode.Name != "" {
				subgraph.Relations = append(subgraph.Relations, types.Relation{
					SourceName: startNode.Name,
					TargetName: endNode.Name,
					Type:       rel.Type,
				})
			}
		}
	}

	for _, entity := range entitiesMap {
		subgraph.Entities = append(subgraph.Entities, entity)
	}

	return subgraph
}