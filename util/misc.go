package util

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func JsonHash(s interface{}) string {
	bs, _ := json.Marshal(s)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MaxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func CopyIntSlice(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}

func CopyStringIntMap(m map[string]int) map[string]int {
	out := make(map[string]int)
	for k, v := range m {
		out[k] = v
	}
	return out
}
