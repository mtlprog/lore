package service

import (
	"strings"
	"testing"

	"github.com/mtlprog/lore/internal/model"
)

// Valid test account IDs (from Stellar SDK test fixtures)
const (
	testAccountID1 = "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7"
	testAccountID2 = "GB7TAYRUZGE6TVT7NHP5SMIZRNQA6PLM423EYISAOAP3MKYIQMVYP2JO"
)

func TestInitXDRBuilder_GenerateParticipantXDR(t *testing.T) {
	builder := NewInitXDRBuilder()

	tests := []struct {
		name        string
		original    model.ParticipantFormData
		current     model.ParticipantFormData
		sequenceNum int64
		wantOps     int
		wantErr     bool
		errContains string
	}{
		{
			name:     "empty account ID",
			original: model.ParticipantFormData{},
			current: model.ParticipantFormData{
				AccountID: "",
				Name:      "Test",
			},
			sequenceNum: 1,
			wantErr:     true,
			errContains: "account ID required",
		},
		{
			name:     "invalid account ID",
			original: model.ParticipantFormData{},
			current: model.ParticipantFormData{
				AccountID: "invalid",
				Name:      "Test",
			},
			sequenceNum: 1,
			wantErr:     true,
			errContains: "invalid account ID",
		},
		{
			name: "no changes",
			original: model.ParticipantFormData{
				AccountID: testAccountID1,
				Name:      "Test",
			},
			current: model.ParticipantFormData{
				AccountID: testAccountID1,
				Name:      "Test",
			},
			sequenceNum: 1,
			wantErr:     true,
			errContains: "no changes",
		},
		{
			name:     "set name",
			original: model.ParticipantFormData{AccountID: testAccountID1},
			current: model.ParticipantFormData{
				AccountID: testAccountID1,
				Name:      "John Doe",
			},
			sequenceNum: 1,
			wantOps:     1,
		},
		{
			name: "delete name",
			original: model.ParticipantFormData{
				AccountID: testAccountID1,
				Name:      "John Doe",
			},
			current:     model.ParticipantFormData{AccountID: testAccountID1},
			sequenceNum: 1,
			wantOps:     1,
		},
		{
			name: "change multiple fields",
			original: model.ParticipantFormData{
				AccountID: testAccountID1,
				Name:      "Old Name",
			},
			current: model.ParticipantFormData{
				AccountID: testAccountID1,
				Name:      "New Name",
				About:     "About text",
				Website:   "https://example.com",
			},
			sequenceNum: 1,
			wantOps:     3,
		},
		{
			name:     "add tags",
			original: model.ParticipantFormData{AccountID: testAccountID1},
			current: model.ParticipantFormData{
				AccountID: testAccountID1,
				Tags:      []string{"Developer", "Crypto"},
			},
			sequenceNum: 1,
			wantOps:     2,
		},
		{
			name: "remove tag",
			original: model.ParticipantFormData{
				AccountID: testAccountID1,
				Tags:      []string{"Developer"},
			},
			current: model.ParticipantFormData{
				AccountID: testAccountID1,
				Tags:      []string{},
			},
			sequenceNum: 1,
			wantOps:     1,
		},
		{
			name:     "invalid tags are filtered",
			original: model.ParticipantFormData{AccountID: testAccountID1},
			current: model.ParticipantFormData{
				AccountID: testAccountID1,
				Tags:      []string{"Developer", "InvalidTag123"},
			},
			sequenceNum: 1,
			wantOps:     1, // Only Developer is valid
		},
		{
			name:     "add PartOf relation",
			original: model.ParticipantFormData{AccountID: testAccountID1},
			current: model.ParticipantFormData{
				AccountID: testAccountID1,
				PartOf: []model.NumberedField{
					{Index: "001", Value: testAccountID2},
				},
			},
			sequenceNum: 1,
			wantOps:     1,
		},
		{
			name: "remove PartOf relation",
			original: model.ParticipantFormData{
				AccountID: testAccountID1,
				PartOf: []model.NumberedField{
					{Index: "001", Value: testAccountID2},
				},
			},
			current:     model.ParticipantFormData{AccountID: testAccountID1},
			sequenceNum: 1,
			wantOps:     1,
		},
		{
			name:     "invalid PartOf account ID is skipped",
			original: model.ParticipantFormData{AccountID: testAccountID1},
			current: model.ParticipantFormData{
				AccountID: testAccountID1,
				PartOf: []model.NumberedField{
					{Index: "001", Value: "invalid-account"},
				},
			},
			sequenceNum: 1,
			wantErr:     true, // No valid changes
			errContains: "no changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdr, ops, err := builder.GenerateParticipantXDR(tt.original, tt.current, tt.sequenceNum)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(ops) != tt.wantOps {
				t.Errorf("got %d operations, want %d", len(ops), tt.wantOps)
			}

			if xdr == "" {
				t.Error("expected non-empty XDR")
			}
		})
	}
}

func TestInitXDRBuilder_GenerateCorporateXDR(t *testing.T) {
	builder := NewInitXDRBuilder()

	tests := []struct {
		name        string
		original    model.CorporateFormData
		current     model.CorporateFormData
		sequenceNum int64
		wantOps     int
		wantErr     bool
	}{
		{
			name:     "set company name",
			original: model.CorporateFormData{AccountID: testAccountID1},
			current: model.CorporateFormData{
				AccountID: testAccountID1,
				Name:      "Company Inc.",
			},
			sequenceNum: 1,
			wantOps:     1,
		},
		{
			name:     "add MyPart member",
			original: model.CorporateFormData{AccountID: testAccountID1},
			current: model.CorporateFormData{
				AccountID: testAccountID1,
				MyPart: []model.NumberedField{
					{Index: "001", Value: testAccountID2},
				},
			},
			sequenceNum: 1,
			wantOps:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdr, ops, err := builder.GenerateCorporateXDR(tt.original, tt.current, tt.sequenceNum)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(ops) != tt.wantOps {
				t.Errorf("got %d operations, want %d", len(ops), tt.wantOps)
			}

			if xdr == "" {
				t.Error("expected non-empty XDR")
			}
		})
	}
}

func TestBuildLabLink(t *testing.T) {
	xdr := "AAAAAgAAAADYP1UCGH4x"

	link := BuildLabLink(xdr)

	if link == "" {
		t.Error("expected non-empty link")
	}

	// Should contain the XDR URL-encoded
	if !strings.Contains(link, "xdr=") {
		t.Error("link should contain xdr parameter")
	}

	// Should target lab.stellar.org
	if !strings.Contains(link, "lab.stellar.org") {
		t.Error("link should target lab.stellar.org")
	}

	// Should use cli-sign endpoint
	if !strings.Contains(link, "/transaction/cli-sign") {
		t.Error("link should use cli-sign endpoint")
	}
}

func TestDiffTags_FiltersInvalidTags(t *testing.T) {
	builder := NewInitXDRBuilder()

	// Test with mix of valid and invalid tags
	ops, summaries := builder.diffTags(testAccountID1,
		[]string{"ValidButNotInList", "Developer"},  // original: Developer is valid
		[]string{"InvalidTag", "Crypto", "FakeTag"}, // current: Crypto is valid
	)

	// Should have 2 operations: delete Developer, add Crypto
	// Invalid tags should be filtered out
	if len(ops) != 2 {
		t.Errorf("expected 2 operations, got %d", len(ops))
	}

	if len(summaries) != 2 {
		t.Errorf("expected 2 summaries, got %d", len(summaries))
	}

	// Check that the operations are correct
	var hasDeleteDeveloper, hasAddCrypto bool
	for _, s := range summaries {
		if s.Action == "Delete" && s.Key == "TagDeveloper" {
			hasDeleteDeveloper = true
		}
		if s.Action == "Set" && s.Key == "TagCrypto" {
			hasAddCrypto = true
		}
	}

	if !hasDeleteDeveloper {
		t.Error("expected delete operation for TagDeveloper")
	}
	if !hasAddCrypto {
		t.Error("expected set operation for TagCrypto")
	}
}

func TestEncodeDecodeParticipant(t *testing.T) {
	original := model.ParticipantFormData{
		AccountID: testAccountID1,
		Name:      "Test User",
		About:     "About me",
		Website:   "https://example.com",
		PartOf: []model.NumberedField{
			{Index: "001", Value: testAccountID2},
		},
		Tags: []string{"Developer", "Crypto"},
	}

	encoded, err := EncodeOriginalData(original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	decoded, err := DecodeOriginalParticipant(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if decoded.AccountID != original.AccountID {
		t.Errorf("AccountID mismatch: got %q, want %q", decoded.AccountID, original.AccountID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.About != original.About {
		t.Errorf("About mismatch: got %q, want %q", decoded.About, original.About)
	}
	if decoded.Website != original.Website {
		t.Errorf("Website mismatch: got %q, want %q", decoded.Website, original.Website)
	}
	if len(decoded.PartOf) != len(original.PartOf) {
		t.Errorf("PartOf length mismatch: got %d, want %d", len(decoded.PartOf), len(original.PartOf))
	}
	if len(decoded.Tags) != len(original.Tags) {
		t.Errorf("Tags length mismatch: got %d, want %d", len(decoded.Tags), len(original.Tags))
	}
}

func TestEncodeDecodeCorporate(t *testing.T) {
	original := model.CorporateFormData{
		AccountID: testAccountID1,
		Name:      "Company Inc.",
		About:     "About company",
		Website:   "https://company.com",
		MyPart: []model.NumberedField{
			{Index: "001", Value: testAccountID2},
		},
		Tags: []string{"Business"},
	}

	encoded, err := EncodeOriginalData(original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	decoded, err := DecodeOriginalCorporate(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if decoded.AccountID != original.AccountID {
		t.Errorf("AccountID mismatch: got %q, want %q", decoded.AccountID, original.AccountID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
	}
}

func TestValidateAccountID(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		wantErr   bool
	}{
		{
			name:      "valid account ID",
			accountID: testAccountID1,
			wantErr:   false,
		},
		{
			name:      "empty account ID",
			accountID: "",
			wantErr:   true,
		},
		{
			name:      "invalid format",
			accountID: "invalid",
			wantErr:   true,
		},
		{
			name:      "wrong prefix - secret key",
			accountID: "SBILUHQVXKTLPYXHHBL4IQ7ISJ3AKDTI2ZGU3EXFHDONP6MKRXMKMMGC",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAccountID(tt.accountID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAccountID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
