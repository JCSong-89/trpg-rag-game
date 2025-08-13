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

func Grok3Client(prompt string) (string, error) {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("XAI_API_KEY 환경 변수를 설정해주세요")
	}

	requestPayload := types.GrokHttpRequest{
		Messages: []types.GrokMessage{
			{Role: "user", Content: "Hello, Grok! What's the weather like in Seoul today?"},
		},
		Model:       "grok-3-mini-beta",
		Temperature: 0.7,
		MaxTokens:   1024,
		Stream:      false,
	}

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		log.Fatalf("JSON 마샬링 실패: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("HTTP 요청 생성 실패: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("API 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("응답 바디 읽기 실패: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API 에러: %s, 응답: %s", resp.Status, string(body))
	}

	var apiResponse types.GrokHttpResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Fatalf("JSON 언마샬링 실패: %v", err)
	}

	if len(apiResponse.Choices) > 0 {
		return apiResponse.Choices[0].Message.Content, nil
	}

	failMessage := "Grok으로부터 답변을 받지 못했습니다."
	fmt.Println(failMessage)
	return failMessage, nil
}