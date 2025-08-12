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
		"qdrantId": qdrantId, // Qdrant 벡터 포인터
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
	// ExecuteWrite는 트랜잭션을 자동으로 관리해주는 매우 편리한 함수입니다.
	// 함수 내에서 에러가 발생하면 자동으로 롤백(Rollback)하고, 성공하면 커밋(Commit)합니다.
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Neo4j에 노드 삽입 (qdrantPointID 전달)
		if err := insertNodeToNeo4j(ctx, tx, entity, qdrantPointID); err != nil {
			return nil, err
		}
		// Qdrant에 벡터 삽입
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
		qdrantPointID := uuid.New().String() // Qdrant 전용 ID 생성

		// 임베딩 생성 로직
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

		// ★★★ 이제 이 함수를 호출합니다! ★★★
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

	// ExecuteWrite를 사용하여 전체 관계 삽입을 하나의 트랜잭션으로 묶습니다.
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, rel := range relations {
			safeRelationType := strings.ReplaceAll(rel.Type, " ", "_")
			query := `
                MATCH (a {entityId: $sourceId})
                MATCH (b {entityId: $targetId})
                CREATE (a)-[r:%s]->(b)
                RETURN type(r) AS created_relation_type
            `
			// 관계 타입을 Cypher 쿼리에 안전하게 삽입
			formattedQuery := fmt.Sprintf(query, safeRelationType)

			// tx.Run을 사용하여 쿼리를 실행합니다.
			result, err := tx.Run(ctx, formattedQuery, map[string]any{
				"sourceId": rel.SourceName,
				"targetId": rel.TargetName,
			})
			if err != nil {
				// 트랜잭션 내에서 에러가 발생하면 전체가 롤백됩니다.
				return nil, fmt.Errorf("관계 생성 쿼리 실행 실패 (%s->%s): %w", rel.SourceName, rel.TargetName, err)
			}

			// ▼▼▼ 여기가 핵심적인 수정 부분입니다! ▼▼▼
			// result.Consume()을 호출하여 서버로부터의 모든 결과를 수신하고
			// 스트림을 닫아 해당 쿼리가 완전히 완료되었음을 보장합니다.
			summary, err := result.Consume(ctx)
			if err != nil {
				return nil, fmt.Errorf("결과 소비(Consume) 실패 (%s->%s): %w", rel.SourceName, rel.TargetName, err)
			}

			// 실제로 관계가 생성되었는지 카운터를 통해 확인하고 로그를 남깁니다.
			if summary.Counters().RelationshipsCreated() > 0 {
				log.Printf("... Neo4j에 관계 '%s-[:%s]->%s' 삽입 완료.", rel.SourceName, rel.Type, rel.TargetName)
			} else {
				log.Printf("경고: 관계를 생성하지 못했습니다. 노드를 찾을 수 없음: %s-[:%s]->%s", rel.SourceName, rel.Type, rel.TargetName)
			}
		}
		// 모든 관계 생성이 성공하면 nil을 반환하여 트랜잭션을 커밋합니다.
		return nil, nil
	})

	if err != nil {
		log.Fatalf("관계 삽입 트랜잭션이 최종적으로 실패했습니다: %v", err)
	}
}