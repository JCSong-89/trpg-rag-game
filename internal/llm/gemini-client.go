package llm

import (
	"context"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"os"
)

func GeminiClient(ctx context.Context) (*genai.GenerativeModel, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY 환경 변수를 설정해주세요")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("gemini 클라이언트 생성 실패: %v", err)
	}

	// 사용할 모델 지정
	model := client.GenerativeModel("gemini-1.5-pro-latest")
	return model, nil
}