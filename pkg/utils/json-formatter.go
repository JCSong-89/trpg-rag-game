package utils

import (
	"errors"
	"strings"
)

func ExtractJSONFromString(rawResponse string) (string, error) {
	cleanResponse := strings.Replace(rawResponse, "```json", "", 1)
	cleanResponse = strings.Replace(cleanResponse, "```", "", -1)
	cleanResponse = strings.TrimSpace(cleanResponse)

	startIndex := strings.Index(cleanResponse, "{")
	if startIndex == -1 {
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