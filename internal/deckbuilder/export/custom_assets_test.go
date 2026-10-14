package deckexport

import "testing"

func TestValidateCustomTTSCardBackURL(t *testing.T) {
	valid := "https://custom-assets.casterscompendium.com/assets/tts-card-back/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png"
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "default blank", url: ""},
		{name: "service asset", url: valid},
		{name: "insecure", url: "http" + valid[len("https"):], wantErr: true},
		{name: "other host", url: "https://example.com/assets/tts-card-back/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png", wantErr: true},
		{name: "simulator art", url: "https://custom-assets.casterscompendium.com/assets/simulator-card-art/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png", wantErr: true},
		{name: "caster token", url: "https://custom-assets.casterscompendium.com/assets/simulator-caster-token/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png", wantErr: true},
		{name: "short digest", url: "https://custom-assets.casterscompendium.com/assets/tts-card-back/0123.png", wantErr: true},
		{name: "query", url: valid + "?track=1", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateCustomTTSCardBackURL(test.url)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateCustomTTSCardBackURL(%q) error = %v, wantErr %t", test.url, err, test.wantErr)
			}
			err = ValidateSimulatorCardBackURL(test.url)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateSimulatorCardBackURL(%q) error = %v, wantErr %t", test.url, err, test.wantErr)
			}
		})
	}
}

func TestValidateSimulatorCasterTokenURL(t *testing.T) {
	valid := "https://custom-assets.casterscompendium.com/assets/simulator-caster-token/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png"
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "default blank", url: ""},
		{name: "service asset", url: valid},
		{name: "insecure", url: "http" + valid[len("https"):], wantErr: true},
		{name: "other host", url: "https://example.com/assets/simulator-caster-token/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png", wantErr: true},
		{name: "card back", url: "https://custom-assets.casterscompendium.com/assets/tts-card-back/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png", wantErr: true},
		{name: "simulator art", url: "https://custom-assets.casterscompendium.com/assets/simulator-card-art/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png", wantErr: true},
		{name: "short digest", url: "https://custom-assets.casterscompendium.com/assets/simulator-caster-token/0123.png", wantErr: true},
		{name: "query", url: valid + "?track=1", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateSimulatorCasterTokenURL(test.url)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateSimulatorCasterTokenURL(%q) error = %v, wantErr %t", test.url, err, test.wantErr)
			}
		})
	}
}
