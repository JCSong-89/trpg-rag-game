package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"io"
	"net/http"
	"os"
)

// API Endpoint URL
const httpApiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"

// GenerateContentWithHTTP 함수는 프롬프트를 받아 http 요청을 보내고 결과 텍스트를 반환합니다.
func GenerateContentWithHTTP(ctx context.Context, prompt string) (string, error) {
	// 1. API 키 가져오기
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY 환경 변수를 설정해주세요")
	}

	// 2. 요청 데이터 생성
	payload := types.GeminiHttpRequest{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Parts: []struct {
					Text string `json:"text"`
				}{
					{Text: prompt},
				},
			},
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("JSON 데이터 생성 실패: %v", err)
	}

	// 3. HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "POST", httpApiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("HTTP 요청 객체 생성 실패: %v", err)
	}

	// 4. 헤더 설정
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", apiKey)

	// 5. 요청 실행
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API 요청 실행 실패: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("API 응답 읽기 실패: %v", err)
	}

	// 6. 에러 응답 확인
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API가 에러를 반환했습니다 (상태 코드: %d): %s", resp.StatusCode, string(body))
	}

	// 7. 응답 파싱 및 결과 반환`
	var apiResponse types.GeminiHttpResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("JSON 응답 파싱 실패: %v", err)
	}

	if len(apiResponse.Candidates) > 0 && len(apiResponse.Candidates[0].Content.Parts) > 0 {
		return apiResponse.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("응답에서 텍스트를 찾을 수 없습니다")
}