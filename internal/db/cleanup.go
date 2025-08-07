package db

import (
	"context"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/qdrant/go-client/qdrant"
)

func Cleanup(ctx context.Context, driver neo4j.DriverWithContext, qdrantCollectionsClient qdrant.CollectionsClient, collectionName string) {

	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		tx.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
		return nil, nil
	})

	qdrantCollectionsClient.Delete(ctx, &qdrant.DeleteCollection{CollectionName: collectionName})
}