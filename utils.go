package main

import (
	"errors"
	"math/rand"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// https://play.golang.org/p/Qg_uv_inCek
// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

var (
	tranformator = transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)

	stringLenZero = errors.New("Length array is zero.")
)

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}

func normalizeString(value string) string {
	stringNorm, _, _ := transform.String(tranformator, value)
	return stringNorm
}

func getRandomStringFromArray(arr []string) (string, error) {
	if len(arr) == 0 {
		return "", stringLenZero
	}

	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s) // initialize local pseudorandom generator
	index := r.Intn(len(arr))
	return arr[index], nil
}

func envToBool(env string) bool {
	if env == "" {
		return false
	} else {
		return true
	}
}

func envToArrString(env string) []string {
	return strings.Split(env, ",")
}
