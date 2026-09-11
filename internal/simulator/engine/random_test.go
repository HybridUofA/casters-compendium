package engine

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestSeededRandomProducesRepeatableSequence(t *testing.T) {
	seed := MatchSeed{First: 12345, Second: 67890}
	first := NewSeededRandom(seed)
	second := NewSeededRandom(seed)

	for call := 0; call < 100; call++ {
		upperBound := call%17 + 1
		firstValue := first.RandInt(upperBound)
		secondValue := second.RandInt(upperBound)
		if firstValue != secondValue {
			t.Fatalf("call %d produced %d and %d from the same seed", call, firstValue, secondValue)
		}
	}
}

func TestSeededRandomKeepsValuesWithinBounds(t *testing.T) {
	random := NewSeededRandom(MatchSeed{First: 42, Second: 84})

	for upperBound := 1; upperBound <= 50; upperBound++ {
		for call := 0; call < 20; call++ {
			value := random.RandInt(upperBound)
			if value < 0 || value >= upperBound {
				t.Fatalf("RandInt(%d) = %d; want a value in [0, %d)", upperBound, value, upperBound)
			}
		}
	}
}

func TestSeededRandomMakesShuffleRepeatable(t *testing.T) {
	original := []model.MatchCardID{"a", "b", "c", "d", "e", "f", "g", "h"}
	first := append([]model.MatchCardID(nil), original...)
	second := append([]model.MatchCardID(nil), original...)
	seed := MatchSeed{First: 9876, Second: 5432}

	if err := shuffleDeck(first, NewSeededRandom(seed)); err != nil {
		t.Fatalf("first shuffleDeck() error = %v", err)
	}
	if err := shuffleDeck(second, NewSeededRandom(seed)); err != nil {
		t.Fatalf("second shuffleDeck() error = %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same seed produced different shuffles: %v and %v", first, second)
	}
	if reflect.DeepEqual(first, original) {
		t.Fatalf("test seed left deck unshuffled: %v", first)
	}
}

func TestSeededRandomPanicsWhenUninitialized(t *testing.T) {
	tests := []struct {
		name   string
		random *SeededRandom
	}{
		{name: "nil receiver", random: nil},
		{name: "missing generator", random: &SeededRandom{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertPanics(t, func() {
				test.random.RandInt(1)
			})
		})
	}
}

func TestSeededRandomPanicsForInvalidUpperBound(t *testing.T) {
	for _, upperBound := range []int{0, -1, -100} {
		t.Run(strconv.Itoa(upperBound), func(t *testing.T) {
			random := NewSeededRandom(MatchSeed{First: 1, Second: 2})
			assertPanics(t, func() {
				random.RandInt(upperBound)
			})
		})
	}
}

func assertPanics(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("operation did not panic")
		}
	}()
	operation()
}
