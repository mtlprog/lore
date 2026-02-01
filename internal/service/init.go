package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	"github.com/mtlprog/lore/internal/model"
	"github.com/samber/lo"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
)

// InitXDRBuilder generates Stellar XDR transactions for init forms.
type InitXDRBuilder struct {
	networkPassphrase string
}

// NewInitXDRBuilder creates a new XDR builder for the Stellar public network.
func NewInitXDRBuilder() *InitXDRBuilder {
	return &InitXDRBuilder{
		networkPassphrase: network.PublicNetworkPassphrase,
	}
}

// GenerateParticipantXDR compares original vs current form data and generates XDR.
func (b *InitXDRBuilder) GenerateParticipantXDR(
	original, current model.ParticipantFormData,
	sequenceNum int64,
) (xdr string, ops []model.InitOpSummary, err error) {
	if current.AccountID == "" {
		return "", nil, fmt.Errorf("account ID required")
	}

	if _, err := keypair.ParseAddress(current.AccountID); err != nil {
		return "", nil, fmt.Errorf("invalid account ID: %w", err)
	}

	var operations []txnbuild.Operation

	// Simple fields
	if current.Name != original.Name {
		op, summary := b.manageDataOp("Name", current.Name)
		operations = append(operations, op)
		ops = append(ops, summary)
	}

	if current.About != original.About {
		op, summary := b.manageDataOp("About", current.About)
		operations = append(operations, op)
		ops = append(ops, summary)
	}

	if current.Website != original.Website {
		op, summary := b.manageDataOp("Website", current.Website)
		operations = append(operations, op)
		ops = append(ops, summary)
	}

	// PartOf relations - handle additions, deletions, and changes
	partOfOps, partOfSummaries := b.diffNumberedFields("PartOf", original.PartOf, current.PartOf)
	operations = append(operations, partOfOps...)
	ops = append(ops, partOfSummaries...)

	// Tags - handle additions and deletions
	tagOps, tagSummaries := b.diffTags(current.AccountID, original.Tags, current.Tags)
	operations = append(operations, tagOps...)
	ops = append(ops, tagSummaries...)

	if len(operations) == 0 {
		return "", nil, fmt.Errorf("no changes to submit")
	}

	xdr, err = b.buildXDR(current.AccountID, sequenceNum, operations)
	if err != nil {
		return "", nil, err
	}

	return xdr, ops, nil
}

// GenerateCorporateXDR compares original vs current form data and generates XDR.
func (b *InitXDRBuilder) GenerateCorporateXDR(
	original, current model.CorporateFormData,
	sequenceNum int64,
) (xdr string, ops []model.InitOpSummary, err error) {
	if current.AccountID == "" {
		return "", nil, fmt.Errorf("account ID required")
	}

	if _, err := keypair.ParseAddress(current.AccountID); err != nil {
		return "", nil, fmt.Errorf("invalid account ID: %w", err)
	}

	var operations []txnbuild.Operation

	// Simple fields
	if current.Name != original.Name {
		op, summary := b.manageDataOp("Name", current.Name)
		operations = append(operations, op)
		ops = append(ops, summary)
	}

	if current.About != original.About {
		op, summary := b.manageDataOp("About", current.About)
		operations = append(operations, op)
		ops = append(ops, summary)
	}

	if current.Website != original.Website {
		op, summary := b.manageDataOp("Website", current.Website)
		operations = append(operations, op)
		ops = append(ops, summary)
	}

	// MyPart relations - handle additions, deletions, and changes
	myPartOps, myPartSummaries := b.diffNumberedFields("MyPart", original.MyPart, current.MyPart)
	operations = append(operations, myPartOps...)
	ops = append(ops, myPartSummaries...)

	// Tags - handle additions and deletions
	tagOps, tagSummaries := b.diffTags(current.AccountID, original.Tags, current.Tags)
	operations = append(operations, tagOps...)
	ops = append(ops, tagSummaries...)

	if len(operations) == 0 {
		return "", nil, fmt.Errorf("no changes to submit")
	}

	xdr, err = b.buildXDR(current.AccountID, sequenceNum, operations)
	if err != nil {
		return "", nil, err
	}

	return xdr, ops, nil
}

// manageDataOp creates a ManageData operation with proper encoding.
func (b *InitXDRBuilder) manageDataOp(name, value string) (txnbuild.Operation, model.InitOpSummary) {
	if value == "" {
		// Delete operation
		return &txnbuild.ManageData{
				Name:  name,
				Value: nil,
			}, model.InitOpSummary{
				Action: "Delete",
				Key:    name,
				Value:  "",
			}
	}

	// Set operation - value is stored as-is in Stellar (not base64 encoded by us)
	return &txnbuild.ManageData{
			Name:  name,
			Value: []byte(value),
		}, model.InitOpSummary{
			Action: "Set",
			Key:    name,
			Value:  value,
		}
}

// diffNumberedFields compares original vs current numbered fields and generates operations.
// Preserves original indices when possible, only renumbers new fields without indices.
func (b *InitXDRBuilder) diffNumberedFields(
	prefix string,
	original, current []model.NumberedField,
) ([]txnbuild.Operation, []model.InitOpSummary) {
	var ops []txnbuild.Operation
	var summaries []model.InitOpSummary

	// Collect original keys
	originalKeys := make(map[string]string)
	for _, f := range original {
		if f.Value != "" {
			key := prefix + f.Index
			originalKeys[key] = f.Value
		}
	}

	// Collect current keys, preserving original indices
	currentKeys := make(map[string]string)
	usedIndices := make(map[string]bool)

	for _, f := range current {
		if f.Value == "" {
			continue
		}
		// Validate account ID format
		if _, err := keypair.ParseAddress(f.Value); err != nil {
			continue // Skip invalid account IDs
		}

		index := f.Index
		if index == "" {
			// Find next available index with 3-digit padding
			for i := 1; i <= 999; i++ {
				idx := fmt.Sprintf("%03d", i)
				if !usedIndices[idx] {
					index = idx
					break
				}
			}
		}
		key := prefix + index
		currentKeys[key] = f.Value
		usedIndices[index] = true
	}

	// Delete keys that are no longer present or have changed
	for key, oldVal := range originalKeys {
		if newVal, exists := currentKeys[key]; !exists || newVal != oldVal {
			op, summary := b.manageDataOp(key, "")
			ops = append(ops, op)
			summaries = append(summaries, summary)
		}
	}

	// Add/update current keys
	for key, val := range currentKeys {
		if oldVal, exists := originalKeys[key]; !exists || oldVal != val {
			op, summary := b.manageDataOp(key, val)
			ops = append(ops, op)
			summaries = append(summaries, summary)
		}
	}

	// Sort operations by key for consistent output
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].(*txnbuild.ManageData).Name < ops[j].(*txnbuild.ManageData).Name
	})
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Key < summaries[j].Key
	})

	return ops, summaries
}

// diffTags compares original vs current tags and generates operations.
func (b *InitXDRBuilder) diffTags(
	accountID string,
	original, current []string,
) ([]txnbuild.Operation, []model.InitOpSummary) {
	var ops []txnbuild.Operation
	var summaries []model.InitOpSummary

	// Tags to delete (in original but not in current)
	toDelete := lo.Filter(original, func(tag string, _ int) bool {
		return !lo.Contains(current, tag)
	})

	for _, tag := range toDelete {
		key := "Tag" + tag
		op, summary := b.manageDataOp(key, "")
		ops = append(ops, op)
		summaries = append(summaries, summary)
	}

	// Tags to add (in current but not in original)
	toAdd := lo.Filter(current, func(tag string, _ int) bool {
		return !lo.Contains(original, tag)
	})

	for _, tag := range toAdd {
		key := "Tag" + tag
		// Tag value is the account's own ID
		op, summary := b.manageDataOp(key, accountID)
		ops = append(ops, op)
		summaries = append(summaries, summary)
	}

	return ops, summaries
}

// buildXDR creates an unsigned transaction envelope XDR.
func (b *InitXDRBuilder) buildXDR(accountID string, sequenceNum int64, operations []txnbuild.Operation) (string, error) {
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: accountID,
				Sequence:  sequenceNum,
			},
			Operations:           operations,
			BaseFee:              txnbuild.MinBaseFee,
			Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
			IncrementSequenceNum: false,
		},
	)
	if err != nil {
		return "", fmt.Errorf("build transaction: %w", err)
	}

	xdr, err := tx.Base64()
	if err != nil {
		return "", fmt.Errorf("encode XDR: %w", err)
	}

	return xdr, nil
}

// BuildLabLink creates a Stellar Laboratory link for the given XDR.
func BuildLabLink(xdr string) string {
	// Uses /transaction/cli-sign endpoint (PR #987)
	return "https://lab.stellar.org/transaction/cli-sign?" +
		"networkPassphrase=" + url.QueryEscape(network.PublicNetworkPassphrase) +
		"&xdr=" + url.QueryEscape(xdr)
}

// ParseAccountDataToParticipant converts Horizon account data to ParticipantFormData.
func ParseAccountDataToParticipant(accountID string, data map[string]string) model.ParticipantFormData {
	form := model.ParticipantFormData{
		AccountID: accountID,
		Name:      decodeBase64(data["Name"]),
		About:     decodeBase64(data["About"]),
		Website:   decodeBase64(data["Website"]),
		PartOf:    parseNumberedFields(data, "PartOf"),
		Tags:      parseTagFields(data),
	}
	return form
}

// ParseAccountDataToCorporate converts Horizon account data to CorporateFormData.
func ParseAccountDataToCorporate(accountID string, data map[string]string) model.CorporateFormData {
	form := model.CorporateFormData{
		AccountID: accountID,
		Name:      decodeBase64(data["Name"]),
		About:     decodeBase64(data["About"]),
		Website:   decodeBase64(data["Website"]),
		MyPart:    parseNumberedFields(data, "MyPart"),
		Tags:      parseTagFields(data),
	}
	return form
}

// parseNumberedFields extracts numbered fields like PartOf001, PartOf002 from account data.
func parseNumberedFields(data map[string]string, prefix string) []model.NumberedField {
	var fields []model.NumberedField

	for key, val := range data {
		if len(key) <= len(prefix) {
			continue
		}
		if key[:len(prefix)] != prefix {
			continue
		}

		index := key[len(prefix):]
		decoded := decodeBase64(val)
		if decoded == "" {
			continue
		}

		// Validate it's an account ID (56 chars starting with G)
		if len(decoded) != 56 || decoded[0] != 'G' {
			continue
		}

		fields = append(fields, model.NumberedField{
			Index: index,
			Value: decoded,
		})
	}

	// Sort by index
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Index < fields[j].Index
	})

	return fields
}

// parseTagFields extracts tag names from Tag* keys.
func parseTagFields(data map[string]string) []string {
	var tags []string

	for key := range data {
		if len(key) <= 3 {
			continue
		}
		if key[:3] != "Tag" {
			continue
		}

		tagName := key[3:]
		// Only include known tags
		if lo.Contains(model.AvailableTags, tagName) {
			tags = append(tags, tagName)
		}
	}

	sort.Strings(tags)
	return tags
}

// ValidateAccountID validates a Stellar account ID format.
func ValidateAccountID(accountID string) error {
	if accountID == "" {
		return fmt.Errorf("account ID required")
	}
	if _, err := keypair.ParseAddress(accountID); err != nil {
		return fmt.Errorf("invalid account ID format")
	}
	return nil
}

// encodedParticipant is the JSON structure for encoding ParticipantFormData.
type encodedParticipant struct {
	AccountID string                `json:"a"`
	Name      string                `json:"n"`
	About     string                `json:"ab"`
	Website   string                `json:"w"`
	PartOf    []model.NumberedField `json:"p"`
	Tags      []string              `json:"t"`
}

// encodedCorporate is the JSON structure for encoding CorporateFormData.
type encodedCorporate struct {
	AccountID string                `json:"a"`
	Name      string                `json:"n"`
	About     string                `json:"ab"`
	Website   string                `json:"w"`
	MyPart    []model.NumberedField `json:"m"`
	Tags      []string              `json:"t"`
}

// EncodeOriginalData serializes form data to base64-encoded JSON for hidden field.
func EncodeOriginalData(data interface{}) (string, error) {
	var jsonBytes []byte
	var err error

	switch v := data.(type) {
	case model.ParticipantFormData:
		enc := encodedParticipant{
			AccountID: v.AccountID,
			Name:      v.Name,
			About:     v.About,
			Website:   v.Website,
			PartOf:    v.PartOf,
			Tags:      v.Tags,
		}
		jsonBytes, err = json.Marshal(enc)
	case model.CorporateFormData:
		enc := encodedCorporate{
			AccountID: v.AccountID,
			Name:      v.Name,
			About:     v.About,
			Website:   v.Website,
			MyPart:    v.MyPart,
			Tags:      v.Tags,
		}
		jsonBytes, err = json.Marshal(enc)
	default:
		return "", fmt.Errorf("unsupported form data type")
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal form data: %w", err)
	}
	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}

// DecodeOriginalParticipant deserializes base64-encoded data back to ParticipantFormData.
func DecodeOriginalParticipant(encoded string) (model.ParticipantFormData, error) {
	var form model.ParticipantFormData
	if encoded == "" {
		return form, nil
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return form, fmt.Errorf("invalid original data encoding")
	}

	var enc encodedParticipant
	if err := json.Unmarshal(data, &enc); err != nil {
		return form, fmt.Errorf("invalid original data format: %w", err)
	}

	form.AccountID = enc.AccountID
	form.Name = enc.Name
	form.About = enc.About
	form.Website = enc.Website
	form.PartOf = enc.PartOf
	form.Tags = enc.Tags

	return form, nil
}

// DecodeOriginalCorporate deserializes base64-encoded data back to CorporateFormData.
func DecodeOriginalCorporate(encoded string) (model.CorporateFormData, error) {
	var form model.CorporateFormData
	if encoded == "" {
		return form, nil
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return form, fmt.Errorf("invalid original data encoding")
	}

	var enc encodedCorporate
	if err := json.Unmarshal(data, &enc); err != nil {
		return form, fmt.Errorf("invalid original data format: %w", err)
	}

	form.AccountID = enc.AccountID
	form.Name = enc.Name
	form.About = enc.About
	form.Website = enc.Website
	form.MyPart = enc.MyPart
	form.Tags = enc.Tags

	return form, nil
}
