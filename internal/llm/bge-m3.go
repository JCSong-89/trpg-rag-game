package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const bgeAPIURL = "https://router.huggingface.co/hf-inference/models/BAAI/bge-m3/pipeline/feature-extraction"

// EmbeddingRequest는 API에 보낼 요청 본문 구조체입니다.
type EmbeddingRequest struct {
	Inputs []string `json:"inputs"`
}

func GetBGEEmbeddings(texts []string, apiToken string) ([][]float32, error) {
	// 요청 본문 생성
	reqBody := EmbeddingRequest{
		Inputs: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("JSON 인코딩 실패: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequest("POST", bgeAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 생성 실패: %w", err)
	}

	// 헤더 설정
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	// HTTP 요청 실행
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Hugging Face API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	// API 응답 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		// 에러 응답 본문을 읽어서 로그에 포함하면 디버깅에 도움이 됩니다.
		var errorResponse interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		return nil, fmt.Errorf("API가 에러를 반환했습니다. 상태 코드: %d, 응답: %v", resp.StatusCode, errorResponse)
	}

	// 성공 응답 본문 파싱
	var embeddings [][]float32
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, fmt.Errorf("JSON 응답 디코딩 실패: %w", err)
	}

	return embeddings, nil
}