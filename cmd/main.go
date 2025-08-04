package main

import (
	"context"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/internal/config"
	"github.com/JCSong-89/trpg-rag-game/internal/db"
	"github.com/JCSong-89/trpg-rag-game/internal/llm"
	"github.com/JCSong-89/trpg-rag-game/internal/prompt"
	"github.com/joho/godotenv"
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

	quadrantCollectionClient, _, grpcConn := db.NewQuadrantClient(configData)
	defer grpcConn.Close()
	db.Cleanup(ctx, neo4jDriver, quadrantCollectionClient)

	responseText, err := llm.GenerateContentWithHTTP(ctx, prompt.SystemPromt)
	if err != nil {
		log.Fatalf("API 호출 중 에러 발생: %v", err)
	}

	fmt.Println(responseText)
}