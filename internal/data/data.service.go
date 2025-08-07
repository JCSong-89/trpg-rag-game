package data

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

func insertNodeToNeo4j(ctx context.Context, tx neo4j.ManagedTransaction, entity types.Entity) error {
	params := map[string]any{"id": entity.ID, "name": entity.Name}
	for k, v := range entity.Properties {
		params[k] = v
	}

	_, err := tx.Run(ctx, "UNWIND $props as p CREATE (e:"+entity.Label+") SET e = p", map[string]any{"props": params})
	if err != nil {
		return fmt.Errorf("failed to create Neo4j node for %s: %w", entity.Name, err)
	}
	log.Printf("... Inserted Node '%s' into Neo4j.", entity.Name)
	return nil
}

func upsetVectorToQuadrant(ctx context.Context, qdrantClient qdrant.PointsClient, collectionName string, entity types.Entity) error {
	if entity.Embedding == nil {
		return nil
	}

	isWaitOption := true
	_, err := qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName, Wait: &isWaitOption,
		Points: []*qdrant.PointStruct{
			{
				Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: entity.ID}},
				Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: entity.Embedding}}},
				Payload: map[string]*qdrant.Value{"name": {Kind: &qdrant.Value_StringValue{StringValue: entity.Name}}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert Quadrant point for %s: %w", entity.Name, err)
	}
	log.Printf("... Inserted Vector for '%s' into Quadrant.", entity.Name)
	return nil
}
func processSingleEntity(ctx context.Context, session neo4j.SessionWithContext, quadrantClient qdrant.PointsClient, collectionName string, entity types.Entity) error {

	tx, err := session.BeginTransaction(ctx)
	defer tx.Close(ctx)

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	if err = insertNodeToNeo4j(ctx, tx, entity); err != nil {
		return err
	}
	if err = upsetVectorToQuadrant(ctx, quadrantClient, collectionName, entity); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func ProcessAndStoreEntities(ctx context.Context, driver neo4j.DriverWithContext, quadrantClient qdrant.PointsClient, collectionName string, entities []types.Entity) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)
	hfAPIToken := os.Getenv("HUGGING_TOKEN")

	for _, entity := range entities {
		var propStrings []string
		entity.ID = uuid.New().String()

		for key, value := range entity.Properties {
			propStrings = append(propStrings, fmt.Sprintf("%s: %v", key, value))
		}

		textToEmbed := entity.Name
		if len(propStrings) > 0 {
			textToEmbed += ", " + strings.Join(propStrings, ", ")
		}

		// 2. ⭐️ 임베딩 생성 함수 호출
		embeddings, err := llm.GetBGEEmbeddings([]string{textToEmbed}, hfAPIToken)
		if err != nil {
			log.Printf("ID '%s'의 임베딩 생성 실패: %v", entity.ID, err)
			continue // 이 엔티티는 건너뛰고 계속 진행
		}
		entity.Embedding = embeddings[0]

		if err := processSingleEntity(ctx, session, quadrantClient, collectionName, entity); err != nil {
			log.Printf("ERROR processing entity %s: %v", entity.Name, err)
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

	for _, rel := range relations {
		// Neo4j에 관계(Edge) 저장 🕸️
		cypherQuery := fmt.Sprintf(`
            MATCH (a {name: $sourceName})
            MATCH (b {name: $targetName})
            CREATE (a)-[:%s]->(b)
        `, rel.Type)
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx, cypherQuery, map[string]any{
				"sourceName": rel.SourceName,
				"targetName": rel.TargetName,
			})
			return nil, err
		})
		if err != nil {
			log.Fatalf("Failed to create Neo4j relationship %s-[:%s]->%s: %v", rel.SourceName, rel.Type, rel.TargetName, err)
		}
		log.Printf("... Inserted Edge '%s-[:%s]->%s' into Neo4j.", rel.SourceName, rel.Type, rel.TargetName)
	}
}