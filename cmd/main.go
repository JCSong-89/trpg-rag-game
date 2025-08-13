package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/internal/config"
	"github.com/JCSong-89/trpg-rag-game/internal/db"
	"github.com/JCSong-89/trpg-rag-game/internal/llm"
	"github.com/JCSong-89/trpg-rag-game/internal/prompt"
	"github.com/JCSong-89/trpg-rag-game/internal/service"
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"github.com/JCSong-89/trpg-rag-game/pkg/utils"
	"github.com/joho/godotenv"
	"github.com/qdrant/go-client/qdrant"
	"log"
)

func main() {
	userQuery := "LA FC 회장의 직접적인 설득 외에, 손흥민의 이번 이적 결정에 영향을 미친 가장 중요하고 거시적인 외부 요인은 무엇이었나요?"

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
	collectionName := "football_news"
	db.Cleanup(ctx, neo4jDriver, quadrantCollectionClient, collectionName)

	_, err = quadrantCollectionClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{Config: &qdrant.VectorsConfig_Params{
			Params: &qdrant.VectorParams{Size: 1024, Distance: qdrant.Distance_Cosine},
		}},
	})

	responseText, err := llm.GenerateContentWithHTTP(ctx, prompt.SystemPromt)
	if err != nil {
		log.Fatalf("API 호출 중 에러 발생: %v", err)
	}

	jsonData, err := utils.ExtractJSONFromString(responseText)
	if err != nil {
		log.Fatalf("API 호출 중 에러 발생: %v", err)
	}

	entities, relations, err := service.ParseAndRefineResponse(jsonData)
	if err != nil {
		log.Fatalf("LLM 응답 정제 실패: %v", err)
	}

	/* TODO: 임베딩 과정과 릴레이션 생성은 고루틴으로 돌리는게 좋을 듯 */
	service.ProcessAndStoreEntities(ctx, neo4jDriver, pointsClient, collectionName, entities)
	service.InsertRelations(ctx, neo4jDriver, relations)
	fmt.Println(responseText)

	log.Println("경로 1: LLM 키워드 기반 엔티티 추출 시작...")
	keywordPrompt := fmt.Sprintf(prompt.EntityExtractionPromptTemplate, userQuery)
	keyword, err := llm.GenerateContentWithHTTP(ctx, keywordPrompt)
	if err != nil {
		log.Fatalf("Gemini 엔티티 추출 API 호출 실패: %v", err)
	}

	jsonString, err := utils.ExtractJSONFromString(keyword)
	if err != nil {
		log.Fatalf("응답에서 JSON 추출 실패: %v", err)
	}
	var keywordEntityNames []string
	if err := json.Unmarshal([]byte(jsonString), &keywordEntityNames); err != nil {
		log.Fatalf("JSON 배열 파싱 실패: %v", err)
	}
	log.Printf("키워드 기반 추출 결과: %v", keywordEntityNames)

	log.Println("\n경로 2: Qdrant 의미 기반 엔티티 검색 시작...")
	vectorEntityNames, err := service.FindTopKSimilarEntities(ctx, pointsClient, collectionName, userQuery, 3)
	if err != nil {
		log.Printf("경고: Qdrant 의미 검색 실패: %v", err)
	}

	combinedEntities := make(map[string]bool)
	for _, name := range keywordEntityNames {
		combinedEntities[name] = true
	}
	for _, name := range vectorEntityNames {
		combinedEntities[name] = true
	}

	var finalEntityNames []string
	for name := range combinedEntities {
		finalEntityNames = append(finalEntityNames, name)
	}
	log.Printf("\n통합된 최종 탐색 시작 엔티티: %v", finalEntityNames)

	var allSubgraphs []*types.Subgraph
	for _, entityName := range finalEntityNames {
		log.Printf("'%s' 엔티티에 대한 서브그래프 생성 중...", entityName)

		oneHopSubgraph, err := service.GetOneHopSubgraph(ctx, neo4jDriver, entityName)
		if err != nil {
			log.Printf("경고: '%s'의 OneHop 서브그래프 생성 실패: %v", entityName, err)
		}

		multiHopSubgraph, err := service.GetMultiHopSubgraph(ctx, neo4jDriver, entityName, 10)
		if err != nil {
			log.Printf("경고: '%s'의 MultiHop 서브그래프 생성 실패: %v", entityName, err)
		}

		importanceBasedSubgraph, err := service.GetImportanceBasedSubgraph(ctx, neo4jDriver, entityName, 5)
		if err != nil {
			log.Printf("경고: '%s'의 importBased 서브그래프 생성 실패: %v", entityName, err)
		}

		allSubgraphs = append(allSubgraphs, oneHopSubgraph, multiHopSubgraph, importanceBasedSubgraph)
	}

	fusedSubgraph := service.FuseSubgraph(ctx, allSubgraphs, userQuery)
	contextString := utils.SubgraphToString(fusedSubgraph)
	finalPrompt := fmt.Sprintf(prompt.FinalPromptTemplate, contextString, userQuery)
	answer, err := llm.GenerateContentWithHTTP(ctx, finalPrompt)
	if err != nil {
		log.Fatal("LLM 최종 답변 생성 실패: %w", err)
	}

	fmt.Println("최종 답변:", answer)
}