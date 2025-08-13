package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"net/http"
)

const bgeAPIURL = "https://router.huggingface.co/hf-inference/models/BAAI/bge-m3/pipeline/feature-extraction"

func GetBGEEmbeddings(texts []string, apiToken string) ([][]float32, error) {
	reqBody := types.EmbeddingRequest{
		Inputs: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("JSON 인코딩 실패: %w", err)
	}

	req, err := http.NewRequest("POST", bgeAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 생성 실패: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Hugging Face API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResponse interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		return nil, fmt.Errorf("API가 에러를 반환했습니다. 상태 코드: %d, 응답: %v", resp.StatusCode, errorResponse)
	}

	var embeddings [][]float32
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, fmt.Errorf("JSON 응답 디코딩 실패: %w", err)
	}

	return embeddings, nil
}