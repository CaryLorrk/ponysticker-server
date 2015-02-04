package main

import (
	"bufio"
	"strings"
	"unicode"

	"github.com/carylorrk/go-porterstemmer"
)

func transformQueryText(query string) string {
	letter := make([]string, 0, len(query))
	alpha := make([]rune, 0, len(query))
	num := make([]rune, 0, len(query))
	for _, char := range query {
		letter = append(letter, " ")
		alpha = append(alpha, ' ')
		num = append(num, ' ')
		switch {
		case isAlpha(char):
			alpha[len(alpha)-1] = char
		case unicode.IsDigit(char):
			num[len(num)-1] = char
		case isCJK(char):
			letter[len(letter)-1] = string(char)
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(string(alpha)))
	scanner.Split(bufio.ScanWords)
	result := make([]string, 0, len(query)*3)
	for scanner.Scan() {
		porter := porterstemmer.StemString(scanner.Text())
		result = append(result, toTrigram(porter))
	}
	result = append(result, letter...)
	result = append(result, string(num))
	return strings.Join(result, " ")
}

func toTrigram(text string) string {
	if len(text) < 4 {
		return text
	}

	trigrams := make([]string, 0, len(text))
	for i := 2; i < len(text); i++ {
		trigrams = append(trigrams, text[i-2:i+1])
	}
	return strings.Join(trigrams, " ")
}

func isAlpha(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z')
}

func isCJK(char rune) bool {
	if (char >= 0x4E00 && char <= 0x9FFF) ||
		//CJK Unified Ideographs
		(char >= 0x3400 && char <= 0x4DBF) ||
		//CJK Unified Ideographs Extension A
		(char >= 0x20000 && char <= 0x2A6DF) ||
		//CJK Unified Ideographs Extension B
		(char >= 0x2A700 && char <= 0x2B73F) ||
		//CJK Unified Ideographs Extension C
		(char >= 0x2B740 && char <= 0x2B81F) ||
		//CJK Unified Ideographs Extension D
		(char >= 0x2E80 && char <= 0x2EFF) ||
		//CJK Radicals Supplement
		(char >= 0x2F00 && char <= 0x2FDF) ||
		//Kangxi Radicals
		(char >= 0x2FF0 && char <= 0x2FFF) ||
		//Ideographic Description Characters
		(char >= 0x3000 && char <= 0x303F) ||
		//CJK Symbols and Punctuation
		(char >= 0x31C0 && char <= 0x31EF) ||
		//CJK Strokes
		(char >= 0x3200 && char <= 0x32FF) ||
		//Enclosed CJK Letters and Months
		(char >= 0x3300 && char <= 0x33FF) ||
		//CJK Compatibility
		(char >= 0xF900 && char <= 0xFAFF) ||
		//CJK Compatibility Ideograpsh
		(char >= 0xFE30 && char <= 0xFE4F) ||
		//CJK Compatibility Forms
		(char >= 0x2F800 && char <= 0x2FA1F) ||
		//CJK Compatibility Ideographs Supplement
		(char >= 0x3040 && char <= 0x309F) ||
		// Hiragana
		(char >= 0x30A0 && char <= 0x30FF) ||
		// Katakana
		(char >= 0xAC00 && char <= 0xD7A3) {
		// Hangul Syllables
		return true
	}
	return false
}
