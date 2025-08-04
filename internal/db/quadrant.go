package db

import (
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

func NewQuadrantClient(cfg types.Config) (qdrant.CollectionsClient, qdrant.PointsClient, *grpc.ClientConn) {
	conn, err := grpc.Dial(cfg.Db.QuadrantUrI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Quadrant: %v", err)
	}
	pointsClient := qdrant.NewPointsClient(conn)
	quadrantCollectionsClient := qdrant.NewCollectionsClient(conn)

	return quadrantCollectionsClient, pointsClient, conn
}