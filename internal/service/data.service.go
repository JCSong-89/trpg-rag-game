package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/internal/llm"
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/qdrant/go-client/qdrant"
	"log"
	"os"
	"strings"
)

func insertNodeToNeo4j(ctx context.Context, tx neo4j.ManagedTransaction, entity types.Entity, qdrantId string) error {
	params := map[string]any{
		"entityId": entity.ID,
		"name":     entity.Name,
		"qdrantId": qdrantId,
	}
	for k, v := range entity.Properties {
		params[k] = v
	}

	safeLabel := strings.ReplaceAll(entity.Label, " ", "_")
	query := fmt.Sprintf("CREATE (e:%s) SET e = $props", safeLabel)

	_, err := tx.Run(ctx, query, map[string]any{"props": params})
	if err != nil {
		return fmt.Errorf("Neo4j 노드 생성 실패 (%s): %w", entity.Name, err)
	}
	return nil
}

func upsetVectorToQuadrant(ctx context.Context, qdrantClient qdrant.PointsClient, collectionName string, entity types.Entity, pointID string) error {
	if entity.Embedding == nil {
		return nil
	}
	isWaitOption := true
	_, err := qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName, Wait: &isWaitOption,
		Points: []*qdrant.PointStruct{
			{
				Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: pointID}},
				Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: entity.Embedding}}},
				Payload: map[string]*qdrant.Value{"name": {Kind: &qdrant.Value_StringValue{StringValue: entity.Name}}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Quadrant 포인트 업서트 실패 (%s): %w", entity.Name, err)
	}
	return nil
}

func processSingleEntity(ctx context.Context, session neo4j.SessionWithContext, quadrantClient qdrant.PointsClient, collectionName string, entity types.Entity, qdrantPointID string) error {

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		if err := insertNodeToNeo4j(ctx, tx, entity, qdrantPointID); err != nil {
			return nil, err
		}

		if err := upsetVectorToQuadrant(ctx, quadrantClient, collectionName, entity, qdrantPointID); err != nil {
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("단일 엔티티 처리 트랜잭션 실패 (%s): %w", entity.Name, err)
	}

	log.Printf("... 엔티티 '%s' 처리 완료 (Neo4j & Qdrant)", entity.Name)
	return nil
}

func ProcessAndStoreEntities(ctx context.Context, driver neo4j.DriverWithContext, quadrantClient qdrant.PointsClient, collectionName string, entities []types.Entity) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)
	hfAPIToken := os.Getenv("HUGGING_TOKEN")

	for _, entity := range entities {
		qdrantPointID := uuid.New().String()

		var propStrings []string
		for key, value := range entity.Properties {
			propStrings = append(propStrings, fmt.Sprintf("%s: %v", key, value))
		}

		textToEmbed := entity.Name
		if len(propStrings) > 0 {
			textToEmbed += ", " + strings.Join(propStrings, ", ")
		}

		embeddings, err := llm.GetBGEEmbeddings([]string{textToEmbed}, hfAPIToken)
		if err != nil {
			log.Printf("경고: '%s'의 임베딩 생성 실패: %v", entity.ID, err)
			continue
		}
		entity.Embedding = embeddings[0]

		if err := processSingleEntity(ctx, session, quadrantClient, collectionName, entity, qdrantPointID); err != nil {
			log.Printf("에러: 엔티티 처리 중 최종 오류 발생 (%s): %v", entity.Name, err)
		}
	}
}

func ParseAndRefineResponse(jsonString string) ([]types.Entity, []types.Relation, error) {
	var parsedResult types.ParsedData
	if err := json.Unmarshal([]byte(jsonString), &parsedResult); err != nil {
		return nil, nil, fmt.Errorf("JSON 응답 파싱 실패: %w", err)
	}

	return parsedResult.Entities, parsedResult.Relations, nil
}
func InsertRelations(ctx context.Context, driver neo4j.DriverWithContext, relations []types.Relation) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, rel := range relations {
			safeRelationType := strings.ReplaceAll(rel.Type, " ", "_")
			query := `
                MATCH (a {entityId: $sourceId})
                MATCH (b {entityId: $targetId})
                CREATE (a)-[r:%s]->(b)
                RETURN type(r) AS created_relation_type
            `
			formattedQuery := fmt.Sprintf(query, safeRelationType)

			result, err := tx.Run(ctx, formattedQuery, map[string]any{
				"sourceId": rel.SourceName,
				"targetId": rel.TargetName,
			})
			if err != nil {
				// 트랜잭션 내에서 에러가 발생하면 전체가 롤백됩니다.
				return nil, fmt.Errorf("관계 생성 쿼리 실행 실패 (%s->%s): %w", rel.SourceName, rel.TargetName, err)
			}

			summary, err := result.Consume(ctx)
			if err != nil {
				return nil, fmt.Errorf("결과 소비(Consume) 실패 (%s->%s): %w", rel.SourceName, rel.TargetName, err)
			}

			if summary.Counters().RelationshipsCreated() > 0 {
				log.Printf("... Neo4j에 관계 '%s-[:%s]->%s' 삽입 완료.", rel.SourceName, rel.Type, rel.TargetName)
			} else {
				log.Printf("경고: 관계를 생성하지 못했습니다. 노드를 찾을 수 없음: %s-[:%s]->%s", rel.SourceName, rel.Type, rel.TargetName)
			}
		}

		return nil, nil
	})

	if err != nil {
		log.Fatalf("관계 삽입 트랜잭션이 최종적으로 실패했습니다: %v", err)
	}
}