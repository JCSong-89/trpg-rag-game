package service

import (
	"context"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/internal/llm"
	"github.com/qdrant/go-client/qdrant"
	"log"
	"os"
)

func FindTopKSimilarEntities(ctx context.Context, qdrantPointsClient qdrant.PointsClient, collectionName string, query string, topK uint64) ([]string, error) {
	hfAPIToken := os.Getenv("HUGGING_TOKEN")

	// 1. 사용자 질문(Query)을 벡터로 변환
	queryEmbedding, err := llm.GetBGEEmbeddings([]string{query}, hfAPIToken)
	if err != nil || len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("질문 임베딩 생성 실패: %w", err)
	}

	// 2. Qdrant에 검색 요청
	searchResult, err := qdrantPointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Vector:         queryEmbedding[0],
		Limit:          topK,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("Qdrant 벡터 검색 실패: %w", err)
	}

	// 3. 검색 결과에서 엔티티 이름 추출
	var similarEntityNames []string
	for _, point := range searchResult.GetResult() {
		if nameVal, ok := point.GetPayload()["name"]; ok {
			similarEntityNames = append(similarEntityNames, nameVal.GetStringValue())
		}
	}

	log.Printf("Qdrant 의미 검색 완료: %v", similarEntityNames)
	return similarEntityNames, nil
}