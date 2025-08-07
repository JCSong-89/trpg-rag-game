package utils

import (
	"errors"
	"strings"
)

func ExtractJSONFromString(rawResponse string) (string, error) {
	startIndex := strings.Index(rawResponse, "{")
	if startIndex == -1 {
		return "", errors.New("JSON 시작 부분 '{'을 찾을 수 없습니다")
	}

	endIndex := strings.LastIndex(rawResponse, "}")
	if endIndex == -1 {
		return "", errors.New("JSON 끝 부분 '}'을 찾을 수 없습니다")
	}

	if endIndex < startIndex {
		return "", errors.New("JSON 구조가 유효하지 않습니다 (end '}' before start '{')")
	}

	return rawResponse[startIndex : endIndex+1], nil
}