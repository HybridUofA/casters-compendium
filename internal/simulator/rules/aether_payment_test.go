package rules

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestValidateAetherPaymentAcceptsEveryRequiredElement(t *testing.T) {
	pool := model.AetherPool{
		Aes: 3, Aqua: 3, Ignus: 3, Luna: 3,
		Silva: 3, Solis: 3, Terra: 3, Void: 3,
		NonElemental: 3,
	}
	tests := []struct {
		name    string
		element model.Element
		pay     model.AetherPayment
	}{
		{name: "Aes", element: model.ElementAes, pay: model.AetherPayment{Aes: 1, NonElemental: 2}},
		{name: "Aqua", element: model.ElementAqua, pay: model.AetherPayment{Aqua: 1, NonElemental: 2}},
		{name: "Ignus", element: model.ElementIgnus, pay: model.AetherPayment{Ignus: 1, NonElemental: 2}},
		{name: "Luna", element: model.ElementLuna, pay: model.AetherPayment{Luna: 1, NonElemental: 2}},
		{name: "Silva", element: model.ElementSilva, pay: model.AetherPayment{Silva: 1, NonElemental: 2}},
		{name: "Solis", element: model.ElementSolis, pay: model.AetherPayment{Solis: 1, NonElemental: 2}},
		{name: "Terra", element: model.ElementTerra, pay: model.AetherPayment{Terra: 1, NonElemental: 2}},
		{name: "Void", element: model.ElementVoid, pay: model.AetherPayment{Void: 1, NonElemental: 2}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if err := ValidateAetherPayment(pool, testCase.pay, 3, testCase.element); err != nil {
				t.Fatalf("ValidateAetherPayment() error = %v", err)
			}
		})
	}
}

func TestValidateAetherPaymentAcceptsZeroCostWithoutAether(t *testing.T) {
	pool := model.AetherPool{}
	payment := model.AetherPayment{}

	if err := ValidateAetherPayment(pool, payment, 0, model.ElementAes); err != nil {
		t.Fatalf("ValidateAetherPayment() error = %v; want nil", err)
	}
}

func TestValidateAetherPaymentRejectsInvalidPayment(t *testing.T) {
	pool := model.AetherPool{
		Aes: 2, Aqua: 2, Ignus: 2, Luna: 2,
		Silva: 2, Solis: 2, Terra: 2, Void: 2,
		NonElemental: 2,
	}
	tests := []struct {
		name        string
		payment     model.AetherPayment
		cost        int
		element     model.Element
		wantErrPart string
	}{
		{
			name:        "payment supplied for zero cost",
			payment:     model.AetherPayment{Aes: 1},
			cost:        0,
			element:     model.ElementAes,
			wantErrPart: "payment total is 1, expected exactly 0",
		},
		{
			name:        "negative printed cost",
			payment:     model.AetherPayment{},
			cost:        -1,
			element:     model.ElementAes,
			wantErrPart: "cost",
		},
		{
			name:        "negative elemental amount",
			payment:     model.AetherPayment{Aes: -1, Aqua: 2},
			cost:        1,
			element:     model.ElementAqua,
			wantErrPart: "Aes payment cannot be negative",
		},
		{
			name:        "negative non-elemental amount",
			payment:     model.AetherPayment{Aes: 2, NonElemental: -1},
			cost:        1,
			element:     model.ElementAes,
			wantErrPart: "NonElemental payment cannot be negative",
		},
		{
			name:        "elemental amount exceeds pool",
			payment:     model.AetherPayment{Aes: 3},
			cost:        3,
			element:     model.ElementAes,
			wantErrPart: "exceeds available amount",
		},
		{
			name:        "non-elemental amount exceeds pool",
			payment:     model.AetherPayment{Aes: 1, NonElemental: 3},
			cost:        4,
			element:     model.ElementAes,
			wantErrPart: "exceeds available amount",
		},
		{
			name:        "underpayment",
			payment:     model.AetherPayment{Aes: 1},
			cost:        2,
			element:     model.ElementAes,
			wantErrPart: "payment total is 1, expected exactly 2",
		},
		{
			name:        "overpayment",
			payment:     model.AetherPayment{Aes: 2},
			cost:        1,
			element:     model.ElementAes,
			wantErrPart: "payment total is 2, expected exactly 1",
		},
		{
			name:        "missing required element",
			payment:     model.AetherPayment{Aqua: 1, NonElemental: 1},
			cost:        2,
			element:     model.ElementAes,
			wantErrPart: "requires 1 Aes",
		},
		{
			name:        "unknown required element",
			payment:     model.AetherPayment{Aes: 1},
			cost:        1,
			element:     model.Element("unknown"),
			wantErrPart: "Unknown element",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateAetherPayment(pool, testCase.payment, testCase.cost, testCase.element)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ValidateAetherPayment() error = %v; want error containing %q", err, testCase.wantErrPart)
			}
		})
	}
}
