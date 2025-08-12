package service

import (
	"context"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"log"
)

func GetOneHopSubgraph(ctx context.Context, driver neo4j.DriverWithContext, entityName string) (*types.Subgraph, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
            MATCH (e {name: $entityName})-[r]-(neighbor)
            RETURN e, r, neighbor
        `
		records, err := tx.Run(ctx, query, map[string]any{"entityName": entityName})
		if err != nil {
			return nil, err
		}
		// 모든 레코드를 수집하여 반환합니다.
		return records.Collect(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("one-hop 서브그래프 생성 실패: %w", err)
	}

	subgraph := &types.Subgraph{}
	entitiesMap := make(map[string]types.Entity)

	records := result.([]*neo4j.Record)

	for _, record := range records {
		startNodeRecord, _ := record.Get("e")
		relationshipRecord, _ := record.Get("r")
		endNodeRecord, _ := record.Get("neighbor")

		startNode := startNodeRecord.(neo4j.Node)
		relationship := relationshipRecord.(neo4j.Relationship)
		endNode := endNodeRecord.(neo4j.Node)

		if _, exists := entitiesMap[startNode.ElementId]; !exists {
			entitiesMap[startNode.ElementId] = types.Entity{
				ID:         startNode.ElementId,
				Name:       startNode.Props["name"].(string),
				Label:      startNode.Labels[0],
				Properties: startNode.Props,
			}
		}

		if _, exists := entitiesMap[endNode.ElementId]; !exists {
			entitiesMap[endNode.ElementId] = types.Entity{
				ID:         endNode.ElementId,
				Name:       endNode.Props["name"].(string),
				Label:      endNode.Labels[0],
				Properties: endNode.Props,
			}
		}

		subgraph.Relations = append(subgraph.Relations, types.Relation{
			SourceName: startNode.Props["name"].(string),
			TargetName: endNode.Props["name"].(string),
			Type:       relationship.Type,
		})
	}

	for _, entity := range entitiesMap {
		subgraph.Entities = append(subgraph.Entities, entity)
	}

	log.Printf("One-hop 서브그래프 생성 완료: %s (엔티티: %d개, 관계: %d개)", entityName, len(subgraph.Entities), len(subgraph.Relations))
	return subgraph, nil
}

func GetMultiHopSubgraph(ctx context.Context, driver neo4j.DriverWithContext, entityName string, maxHops int) (*types.Subgraph, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := fmt.Sprintf(`
            MATCH p=(e {name: $entityName})-[*1..%d]-(neighbor)
            WHERE e <> neighbor
            RETURN p
        `, maxHops)
		records, err := tx.Run(ctx, query, map[string]any{"entityName": entityName})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})

	if err != nil {
		return nil, fmt.Errorf("Multi-hop 서브그래프 생성 실패: %w", err)
	}

	subgraph := parseSubgraphFromRecords(result.([]*neo4j.Record))
	log.Printf("Multi-hop 서브그래프 생성 완료: %s (최대 %d홉, 엔티티: %d개, 관계: %d개)", entityName, maxHops, len(subgraph.Entities), len(subgraph.Relations))
	return subgraph, nil
}

func GetImportanceBasedSubgraph(ctx context.Context, driver neo4j.DriverWithContext, entityName string, topK int) (*types.Subgraph, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	graphName := "gds-temp-graph-" + uuid.New().String()

	defer func() {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx, `
                CALL gds.graph.exists($graphName) YIELD exists
                WHERE exists
                CALL gds.graph.drop($graphName, false) YIELD graphName
                RETURN graphName
            `, map[string]any{"graphName": graphName})
			return nil, err
		})
		if err != nil {
			log.Printf("경고: GDS 임시 그래프 '%s' 정리 실패: %v", graphName, err)
		}
	}()

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `CALL gds.graph.project($graphName, '*', '*')`, map[string]any{"graphName": graphName})
		if err != nil {
			return nil, fmt.Errorf("GDS 그래프 프로젝션 실패: %w", err)
		}

		query := `
            CALL gds.pageRank.stream($graphName)
            YIELD nodeId, score
            WITH gds.util.asNode(nodeId) AS topNode, score
            ORDER BY score DESC
            LIMIT $topK
            WITH COLLECT(topNode.name) AS topKNames

            MATCH (startNode {name: $entityName})

            UNWIND topKNames AS topKName

            MATCH (topNode {name: topKName})
            WHERE startNode <> topNode

            MATCH p = allShortestPaths((startNode)-[*]-(topNode))
            
            WITH topKNames, COLLECT(p) AS paths
            UNWIND paths AS path
            UNWIND nodes(path) AS node
            UNWIND relationships(path) AS rel
            RETURN topKNames, COLLECT(DISTINCT node) AS nodes, COLLECT(DISTINCT rel) AS rels
        `
		params := map[string]any{
			"graphName":  graphName,
			"entityName": entityName,
			"topK":       int64(topK),
		}

		pagerankResult, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		return pagerankResult.Single(ctx)
	})

	if err != nil {
		log.Printf("경고: 중요도 기반 서브그래프 생성 실패: %v", err)
		return &types.Subgraph{}, nil // nil이 아닌 빈 서브그래프를 반환하여 panic 방지
	}

	record := result.(*neo4j.Record)
	topKNamesInterface, _ := record.Get("topKNames")

	var topKNodeNames []string
	if topKNamesInterface != nil {
		for _, nameInterface := range topKNamesInterface.([]interface{}) {
			topKNodeNames = append(topKNodeNames, nameInterface.(string))
		}
	}

	log.Printf("중요도 기반 분석: PageRank Top %d 노드 = %v", topK, topKNodeNames)
	var arrayRecord []*neo4j.Record
	arrayRecord = append(arrayRecord, record)

	subgraph := parseSubgraphFromRecords(arrayRecord)
	log.Printf("중요도 기반 서브그래프 생성 완료: %s (상위 %d개, 엔티티: %d개, 관계: %d개)", entityName, topK, len(subgraph.Entities), len(subgraph.Relations))
	return subgraph, nil
}