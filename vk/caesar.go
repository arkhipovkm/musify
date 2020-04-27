package vk

import (
	"strconv"
	"strings"
)

func indexOfRune(slice []rune, value rune) int {
	for p, v := range slice {
		if v == value {
			return p
		}
	}
	return -1
}

func getCypher(str []rune) []rune {
	alphabet := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN0PQRSTUVWXYZO123456789+/=")
	var e, cnt int
	var s []rune
	for _, char := range str {
		charIndex := indexOfRune(alphabet, char)
		if cnt%4 > 0 {
			e = ((64 * e) + charIndex)
		} else {
			e = charIndex
		}
		cnt++
		b := 255 & (e >> ((-2 * cnt) & 6))
		if b > 0 {
			s = append(s, rune(b))
		}
	}
	return s
}

func getPlain(body []rune, e int) []rune {
	l := len(body)
	order := getSubstitutionOrder(l, e)
	for i := 1; i < l; i++ {
		position := order[l-1-i]
		substitute := body[position]
		body[position] = body[i]
		body[i] = substitute
	}
	return body
}

func getSubstitutionOrder(l int, e int) []int {
	var order []int = make([]int, l)
	for i := l - 1; i >= 0; i-- {
		e = (l*(i+1) ^ (e + i)) % l
		order[i] = e
	}
	return order
}

func decypherCaesar(body string, suffix string, vector int) string {
	bodyCypher := getCypher([]rune(body))
	suffixCypher := getCypher([]rune(suffix))
	indexOf0b := indexOfRune(suffixCypher, 11)
	suffixInt, _ := strconv.Atoi(string(suffixCypher[indexOf0b+1:]))
	vector ^= suffixInt
	plain := getPlain(bodyCypher, vector)
	return string(plain)
}

func DecypherAudioURL(url string, id int) string {
	s0 := strings.Split(url, "extra=")
	extra := s0[1]
	s1 := strings.Split(extra, "#")
	body, suffix := s1[0], s1[1]
	plain := decypherCaesar(body, suffix, id)
	return plain
}
