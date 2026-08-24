package domain

import (
	"strings"
	"testing"
)

func TestNormalizeProfessionalName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantKey   string
		wantValid bool
	}{
		{
			name:      "simple name",
			input:     "Marina Souza",
			wantName:  "Marina Souza",
			wantKey:   "marina souza",
			wantValid: true,
		},
		{
			name:      "extra whitespace",
			input:     "   Carlos   Alberto   da   Silva   ",
			wantName:  "Carlos Alberto da Silva",
			wantKey:   "carlos alberto da silva",
			wantValid: true,
		},
		{
			name:      "unicode characters and accents",
			input:     "Érika Ávila",
			wantName:  "Érika Ávila",
			wantKey:   "érika ávila",
			wantValid: true,
		},
		{
			name:      "empty name",
			input:     "   ",
			wantName:  "",
			wantKey:   "",
			wantValid: false,
		},
		{
			name:      "single character",
			input:     "A",
			wantName:  "A",
			wantKey:   "a",
			wantValid: true,
		},
		{
			name:      "100 runes",
			input:     strings.Repeat("a", 100),
			wantName:  strings.Repeat("a", 100),
			wantKey:   strings.Repeat("a", 100),
			wantValid: true,
		},
		{
			name:      "101 runes exceeds limit",
			input:     strings.Repeat("a", 101),
			wantName:  "",
			wantKey:   "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotKey, gotValid := NormalizeProfessionalName(tt.input)
			if gotValid != tt.wantValid {
				t.Fatalf("NormalizeProfessionalName(%q) valid = %v, want %v", tt.input, gotValid, tt.wantValid)
			}
			if gotName != tt.wantName {
				t.Errorf("NormalizeProfessionalName(%q) name = %q, want %q", tt.input, gotName, tt.wantName)
			}
			if gotKey != tt.wantKey {
				t.Errorf("NormalizeProfessionalName(%q) key = %q, want %q", tt.input, gotKey, tt.wantKey)
			}
		})
	}
}

func TestNormalizeSchedulingReason(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		minLen    int
		maxLen    int
		wantVal   string
		wantValid bool
	}{
		{
			name:      "valid archive reason",
			input:     "  Férias prolongadas  ",
			minLen:    3,
			maxLen:    240,
			wantVal:   "Férias prolongadas",
			wantValid: true,
		},
		{
			name:      "too short archive reason",
			input:     "ab",
			minLen:    3,
			maxLen:    240,
			wantVal:   "",
			wantValid: false,
		},
		{
			name:      "valid anonymize reason",
			input:     "Solicitação LGPD do titular",
			minLen:    5,
			maxLen:    240,
			wantVal:   "Solicitação LGPD do titular",
			wantValid: true,
		},
		{
			name:      "too short anonymize reason",
			input:     "LGPD",
			minLen:    5,
			maxLen:    240,
			wantVal:   "",
			wantValid: false,
		},
		{
			name:      "too long reason",
			input:     strings.Repeat("x", 241),
			minLen:    3,
			maxLen:    240,
			wantVal:   "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotValid := NormalizeSchedulingReason(tt.input, tt.minLen, tt.maxLen)
			if gotValid != tt.wantValid {
				t.Fatalf("NormalizeSchedulingReason(%q, %d, %d) valid = %v, want %v", tt.input, tt.minLen, tt.maxLen, gotValid, tt.wantValid)
			}
			if gotVal != tt.wantVal {
				t.Errorf("NormalizeSchedulingReason(%q) = %q, want %q", tt.input, gotVal, tt.wantVal)
			}
		})
	}
}

func TestValidProfessionalStatus(t *testing.T) {
	if !ValidProfessionalStatus(ProfessionalStatusActive) {
		t.Errorf("expected active to be valid")
	}
	if !ValidProfessionalStatus(ProfessionalStatusArchived) {
		t.Errorf("expected archived to be valid")
	}
	if ValidProfessionalStatus("deleted") {
		t.Errorf("expected deleted to be invalid")
	}
}
