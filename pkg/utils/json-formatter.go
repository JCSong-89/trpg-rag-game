package utils

import (
	"errors"
	"strings"
)

func ExtractJSONFromString(rawResponse string) (string, error) {
	// 먼저 ```json 또는 ``` 와 같은 마크다운 코드 블록을 제거합니다.
	cleanResponse := strings.Replace(rawResponse, "```json", "", 1)
	cleanResponse = strings.Replace(cleanResponse, "```", "", -1)
	cleanResponse = strings.TrimSpace(cleanResponse) // 앞뒤 공백 제거

	// 정리된 문자열에서 JSON 시작과 끝을 찾습니다.
	startIndex := strings.Index(cleanResponse, "{")
	if startIndex == -1 {
		// 객체(Object)가 아니라 배열(Array)로 시작하는 경우도 처리합니다.
		startIndex = strings.Index(cleanResponse, "[")
		if startIndex == -1 {
			return "", errors.New("JSON 시작 부분인 '{' 또는 '['를 찾을 수 없습니다")
		}
	}

	endIndex := strings.LastIndex(cleanResponse, "}")
	if endIndex == -1 {
		endIndex = strings.LastIndex(cleanResponse, "]")
		if endIndex == -1 {
			return "", errors.New("JSON 끝 부분인 '}' 또는 ']'를 찾을 수 없습니다")
		}
	}

	if endIndex < startIndex {
		return "", errors.New("JSON 구조가 유효하지 않습니다 (end before start)")
	}

	return cleanResponse[startIndex : endIndex+1], nil
}