package engine

import (
	"math/rand/v2"
)

type RandomSource interface {
	// RandInt provides a random integer in the defined range [0, upperBound)
	RandInt(upperBound int) int
}

type MatchSeed struct {
	First  uint64
	Second uint64
}

type SeededRandom struct {
	random *rand.Rand
}

func NewSeededRandom(seed MatchSeed) *SeededRandom {
	source := rand.NewPCG(seed.First, seed.Second)
	generator := rand.New(source)
	return &SeededRandom{random: generator}
}

func (random *SeededRandom) RandInt(upperBound int) int {
	if random == nil || random.random == nil {
		panic("seeded random source is not initialized")
	}
	if upperBound <= 0 {
		panic("random upper bound must be positive")
	}
	return random.random.IntN(upperBound)
}

var _ RandomSource = (*SeededRandom)(nil)
