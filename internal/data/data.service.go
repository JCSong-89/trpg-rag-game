package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/qdrant/go-client/qdrant"
	"log"
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

	for _, entity := range entities {
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