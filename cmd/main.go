package main

import (
	"context"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/internal/config"
	"github.com/JCSong-89/trpg-rag-game/internal/data"
	"github.com/JCSong-89/trpg-rag-game/internal/db"
	"github.com/JCSong-89/trpg-rag-game/internal/llm"
	"github.com/JCSong-89/trpg-rag-game/internal/prompt"
	"github.com/joho/godotenv"
	"github.com/qdrant/go-client/qdrant"
	"log"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf(".env 파일을 로드하는 데 실패했습니다: %v", err)
	}

	ctx := context.Background()
	configData := config.LoadConfig()

	neo4jDriver := db.NewNeo4jDriver(configData)
	defer neo4jDriver.Close(ctx)

	quadrantCollectionClient, pointsClient, grpcConn := db.NewQuadrantClient(configData)
	defer grpcConn.Close()
	/*TODO: 데이터 적재 시작 후에 삭제*/
	db.Cleanup(ctx, neo4jDriver, quadrantCollectionClient)

	collectionName := "football_news"
	_, err = quadrantCollectionClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{Config: &qdrant.VectorsConfig_Params{
			Params: &qdrant.VectorParams{Size: 4, Distance: qdrant.Distance_Cosine},
		}},
	})

	responseText, err := llm.GenerateContentWithHTTP(ctx, prompt.SystemPromt)
	if err != nil {
		log.Fatalf("API 호출 중 에러 발생: %v", err)
	}

	entities, _, err := data.ParseAndRefineResponse(responseText)
	if err != nil {
		log.Fatalf("LLM 응답 정제 실패: %v", err)
	}

	data.ProcessAndStoreEntities(ctx, neo4jDriver, pointsClient, collectionName, entities)
	fmt.Println(responseText)
}