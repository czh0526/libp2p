package testutil

import (
	"math/rand"
	"time"
)

var SeededRand *rand.Rand

func init() {
	SeededRand = NewSeededRand(time.Now().UTC().UnixNano())
}

func NewSeededRand(seed int64) *rand.Rand {
	src := rand.NewSource(seed)
	return rand.New(src)
}
