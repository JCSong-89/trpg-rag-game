package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"io"
	"log"
	"net/http"
	"os"
)

// API 엔드포인트 URL
const apiURL = "https://api.x.ai/v1/chat/completions"

// Grok API 요청을 위한 구조체

func Grok3Client(prompt string) (string, error) {
	// 1. 환경변수 또는 직접 API 키 설정
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("XAI_API_KEY 환경 변수를 설정해주세요")
	}

	// 2. API에 보낼 데이터 구성
	requestPayload := types.GrokHttpRequest{
		Messages: []types.GrokMessage{
			{Role: "user", Content: "Hello, Grok! What's the weather like in Seoul today?"},
		},
		Model:       "grok-3-mini-beta", // 사용하려는 모델
		Temperature: 0.7,
		MaxTokens:   1024,
		Stream:      false,
	}

	// Go 구조체를 JSON 데이터로 변환
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		log.Fatalf("JSON 마샬링 실패: %v", err)
	}

	// 3. HTTP 요청 생성
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("HTTP 요청 생성 실패: %v", err)
	}

	// 헤더 설정
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 4. HTTP 클라이언트로 요청 전송
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("API 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	// 5. 응답 바디 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("응답 바디 읽기 실패: %v", err)
	}

	// HTTP 상태 코드가 200 (OK)가 아니면 에러 처리
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API 에러: %s, 응답: %s", resp.Status, string(body))
	}

	// 6. JSON 응답을 Go 구조체로 변환
	var apiResponse types.GrokHttpResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Fatalf("JSON 언마샬링 실패: %v", err)
	}

	// 7. 결과 출력
	if len(apiResponse.Choices) > 0 {
		return apiResponse.Choices[0].Message.Content, nil
	} else {
		failMessage := "Grok으로부터 답변을 받지 못했습니다."
		fmt.Println(failMessage)
		return failMessage, nil
	}
}