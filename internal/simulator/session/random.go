package session

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine"
)

func GenerateMatchSeed() (engine.MatchSeed, error) {
	var data [16]byte
	_, err := rand.Read(data[:])
	if err != nil {
		return engine.MatchSeed{}, fmt.Errorf("error generating seed: %w", err)
	}
	seed := engine.MatchSeed{
		First:  binary.LittleEndian.Uint64(data[0:8]),
		Second: binary.LittleEndian.Uint64(data[8:16]),
	}
	return seed, nil
}
