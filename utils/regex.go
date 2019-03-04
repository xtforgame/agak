package utils

import (
	"regexp"
)

func FindSubmatch(regex *regexp.Regexp, b []byte) []byte {
	var result []byte
	resultIndexArray := regex.FindSubmatchIndex(b)
	if len(resultIndexArray) >= 4 {
		result = b[resultIndexArray[2]:resultIndexArray[3]]
	}
	return result
}

func FindAllSubmatch(regex *regexp.Regexp, b []byte, n int) [][]byte {
	result := [][]byte{}
	resultIndexArrays := regex.FindAllSubmatchIndex(b, n)
	for _, resultIndexArray := range resultIndexArrays {
		if len(resultIndexArray) >= 4 {
			result = append(result, b[resultIndexArray[2]:resultIndexArray[3]])
		}
	}

	return result
}
