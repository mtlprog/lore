package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/mtlprog/lore/internal/model"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stretchr/testify/assert"
)

func TestParseNumberedDataKeys(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]string
		prefix   string
		expected []string
	}{
		{
			name:     "empty data",
			data:     map[string]string{},
			prefix:   "Website",
			expected: []string{},
		},
		{
			name: "single key without number",
			data: map[string]string{
				"Website": encode("https://example.com"),
			},
			prefix:   "Website",
			expected: []string{"https://example.com"},
		},
		{
			name: "numbered keys",
			data: map[string]string{
				"Website":  encode("https://first.com"),
				"Website1": encode("https://second.com"),
				"Website2": encode("https://third.com"),
			},
			prefix:   "Website",
			expected: []string{"https://first.com", "https://second.com", "https://third.com"},
		},
		{
			name: "numbered keys with leading zeros",
			data: map[string]string{
				"Website":     encode("https://zero.com"),
				"Website0001": encode("https://one.com"),
				"Website0002": encode("https://two.com"),
			},
			prefix:   "Website",
			expected: []string{"https://zero.com", "https://one.com", "https://two.com"},
		},
		{
			name: "out of order keys",
			data: map[string]string{
				"Website3": encode("https://three.com"),
				"Website1": encode("https://one.com"),
				"Website":  encode("https://zero.com"),
				"Website2": encode("https://two.com"),
			},
			prefix:   "Website",
			expected: []string{"https://zero.com", "https://one.com", "https://two.com", "https://three.com"},
		},
		{
			name: "mixed keys with different prefixes",
			data: map[string]string{
				"Website":  encode("https://web.com"),
				"Website1": encode("https://web1.com"),
				"Name":     encode("Test Name"),
				"About":    encode("Test About"),
			},
			prefix:   "Website",
			expected: []string{"https://web.com", "https://web1.com"},
		},
		{
			name: "skip empty values",
			data: map[string]string{
				"Website":  encode("https://valid.com"),
				"Website1": "",
				"Website2": encode("https://another.com"),
			},
			prefix:   "Website",
			expected: []string{"https://valid.com", "https://another.com"},
		},
		{
			name: "skip invalid base64",
			data: map[string]string{
				"Website":  encode("https://valid.com"),
				"Website1": "not-valid-base64!!!",
				"Website2": encode("https://another.com"),
			},
			prefix:   "Website",
			expected: []string{"https://valid.com", "https://another.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNumberedDataKeys(tt.data, tt.prefix)
			if len(tt.expected) == 0 {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseTagKeys(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]string
		expected []string
	}{
		{
			name:     "empty data",
			data:     map[string]string{},
			expected: nil,
		},
		{
			name: "single tag",
			data: map[string]string{
				"TagBelgrade": encode("GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"),
			},
			expected: []string{"Belgrade"},
		},
		{
			name: "multiple tags",
			data: map[string]string{
				"TagBelgrade":   encode("GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"),
				"TagProgrammer": encode("GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA"),
				"TagMontenegro": encode("GTEST1234567890"),
			},
			expected: []string{"Belgrade", "Montenegro", "Programmer"},
		},
		{
			name: "mixed keys with non-tag data",
			data: map[string]string{
				"TagBelgrade": encode("GACCOUNT"),
				"Name":        encode("Test Name"),
				"About":       encode("Test About"),
				"Website":     encode("https://example.com"),
			},
			expected: []string{"Belgrade"},
		},
		{
			name: "Tag key without suffix is ignored",
			data: map[string]string{
				"Tag": encode("GACCOUNT"),
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTagKeys(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecodeBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "valid base64",
			input:    encode("Hello, World!"),
			expected: "Hello, World!",
		},
		{
			name:     "invalid base64",
			input:    "not-valid!!!",
			expected: "",
		},
		{
			name:     "whitespace trimmed",
			input:    encode("  test  "),
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeBase64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func TestFindAssetBalance(t *testing.T) {
	balances := []horizon.Balance{
		{
			Balance: "100.0000000",
			Asset: base.Asset{
				Type:   "native",
				Code:   "",
				Issuer: "",
			},
		},
		{
			Balance: "50.0000000",
			Asset: base.Asset{
				Type:   "credit_alphanum4",
				Code:   "MTLAP",
				Issuer: "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
			},
		},
		{
			Balance: "25.0000000",
			Asset: base.Asset{
				Type:   "credit_alphanum4",
				Code:   "USDC",
				Issuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
			},
		},
	}

	tests := []struct {
		name     string
		code     string
		issuer   string
		expected string
	}{
		{
			name:     "find MTLAP",
			code:     "MTLAP",
			issuer:   "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
			expected: "50.0000000",
		},
		{
			name:     "find USDC",
			code:     "USDC",
			issuer:   "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
			expected: "25.0000000",
		},
		{
			name:     "asset not found",
			code:     "BTC",
			issuer:   "GISSUER",
			expected: "0",
		},
		{
			name:     "wrong issuer",
			code:     "MTLAP",
			issuer:   "GWRONGISSUER",
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findAssetBalance(balances, tt.code, tt.issuer)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindAssetBalance_EmptyBalances(t *testing.T) {
	result := findAssetBalance([]horizon.Balance{}, "MTLAP", "GISSUER")
	assert.Equal(t, "0", result)
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "horizon 404 error",
			err: &horizonclient.Error{
				Response: &http.Response{
					StatusCode: 404,
				},
			},
			expected: true,
		},
		{
			name: "horizon 500 error",
			err: &horizonclient.Error{
				Response: &http.Response{
					StatusCode: 500,
				},
			},
			expected: false,
		},
		{
			name: "horizon error without response",
			err: &horizonclient.Error{
				Response: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewStellarService(t *testing.T) {
	t.Run("creates service with custom URL", func(t *testing.T) {
		s := NewStellarService("https://custom-horizon.example.com")
		assert.NotNil(t, s)
		assert.NotNil(t, s.client)
		assert.Equal(t, "https://custom-horizon.example.com", s.client.HorizonURL)
	})

	t.Run("creates service with default URL", func(t *testing.T) {
		s := NewStellarService("https://horizon.stellar.org")
		assert.NotNil(t, s)
		assert.Equal(t, "https://horizon.stellar.org", s.client.HorizonURL)
	})
}

func TestIsSpamOperation(t *testing.T) {
	tests := []struct {
		name string
		op   model.Operation
		want bool
	}{
		{
			name: "create_claimable_balance is spam",
			op:   model.Operation{Type: "create_claimable_balance"},
			want: true,
		},
		{
			name: "claim_claimable_balance is spam",
			op:   model.Operation{Type: "claim_claimable_balance"},
			want: true,
		},
		{
			name: "small XLM payment (0.5) is spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "0.5"},
			want: true,
		},
		{
			name: "very small XLM payment (0.0001) is spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "0.0001"},
			want: true,
		},
		{
			name: "exactly 1 XLM is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "1.0"},
			want: false,
		},
		{
			name: "exactly 1 XLM integer is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "1"},
			want: false,
		},
		{
			name: "large XLM payment is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "100.5"},
			want: false,
		},
		{
			name: "small non-XLM payment is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "USDC", Amount: "0.5"},
			want: false,
		},
		{
			name: "small MTLAP payment is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "MTLAP", Amount: "0.001"},
			want: false,
		},
		{
			name: "manage_data is NOT spam",
			op:   model.Operation{Type: "manage_data", DataName: "test"},
			want: false,
		},
		{
			name: "change_trust is NOT spam",
			op:   model.Operation{Type: "change_trust", AssetCode: "MTLAP"},
			want: false,
		},
		{
			name: "invalid amount string is NOT spam (parse fails)",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "invalid"},
			want: false,
		},
		{
			name: "empty amount is NOT spam (parse fails)",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: ""},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSpamOperation(tt.op)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetOperationDirection(t *testing.T) {
	const viewer = "GVIEWER"
	const other = "GOTHER"

	tests := []struct {
		name             string
		op               model.Operation
		wantDirection    model.OperationDirection
		wantCounterparty string
	}{
		{
			name:             "payment where viewer is sender",
			op:               model.Operation{Type: "payment", From: viewer, To: other},
			wantDirection:    model.DirectionOut,
			wantCounterparty: other,
		},
		{
			name:             "payment where viewer is recipient",
			op:               model.Operation{Type: "payment", From: other, To: viewer},
			wantDirection:    model.DirectionIn,
			wantCounterparty: other,
		},
		{
			name:             "payment where viewer is both sides is self",
			op:               model.Operation{Type: "payment", From: viewer, To: viewer},
			wantDirection:    model.DirectionSelf,
			wantCounterparty: "",
		},
		{
			name:             "payment where viewer is neither side",
			op:               model.Operation{Type: "payment", From: "GA", To: "GB"},
			wantDirection:    model.DirectionNone,
			wantCounterparty: "",
		},
		{
			name:             "create_account viewer as funder is out",
			op:               model.Operation{Type: "create_account", From: viewer, To: other},
			wantDirection:    model.DirectionOut,
			wantCounterparty: other,
		},
		{
			name:             "create_account viewer as new account is in",
			op:               model.Operation{Type: "create_account", From: other, To: viewer},
			wantDirection:    model.DirectionIn,
			wantCounterparty: other,
		},
		{
			name:             "path_payment_strict_send viewer as sender",
			op:               model.Operation{Type: "path_payment_strict_send", From: viewer, To: other},
			wantDirection:    model.DirectionOut,
			wantCounterparty: other,
		},
		{
			name:             "path_payment_strict_receive viewer as recipient",
			op:               model.Operation{Type: "path_payment_strict_receive", From: other, To: viewer},
			wantDirection:    model.DirectionIn,
			wantCounterparty: other,
		},
		{
			name:             "account_merge viewer as source is out",
			op:               model.Operation{Type: "account_merge", From: viewer, To: other},
			wantDirection:    model.DirectionOut,
			wantCounterparty: other,
		},
		{
			name:             "manage_data is not payment-like",
			op:               model.Operation{Type: "manage_data", DataName: "foo"},
			wantDirection:    model.DirectionNone,
			wantCounterparty: "",
		},
		{
			name:             "change_trust is not payment-like",
			op:               model.Operation{Type: "change_trust", AssetCode: "MTLAP"},
			wantDirection:    model.DirectionNone,
			wantCounterparty: "",
		},
		{
			name:             "manage_sell_offer is not payment-like",
			op:               model.Operation{Type: "manage_sell_offer", Selling: "XLM", Buying: "USDC"},
			wantDirection:    model.DirectionNone,
			wantCounterparty: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := tt.op
			setOperationDirection(&op, viewer)
			assert.Equal(t, tt.wantDirection, op.Direction)
			assert.Equal(t, tt.wantCounterparty, op.Counterparty)
			assert.True(t, op.Direction.IsValid())
		})
	}
}

func TestOperationDirectionIsValid(t *testing.T) {
	assert.True(t, model.DirectionNone.IsValid())
	assert.True(t, model.DirectionIn.IsValid())
	assert.True(t, model.DirectionOut.IsValid())
	assert.True(t, model.DirectionSelf.IsValid())
	assert.False(t, model.OperationDirection("bogus").IsValid())
	assert.False(t, model.OperationDirection("IN").IsValid())
}

// TestDirectionConstantValues pins the string values used across layers (Go code, template
// equality checks, CSS class names). A rename without updating all three places would silently
// break the UI; this test catches it.
func TestDirectionConstantValues(t *testing.T) {
	assert.Equal(t, "", string(model.DirectionNone))
	assert.Equal(t, "in", string(model.DirectionIn))
	assert.Equal(t, "out", string(model.DirectionOut))
	assert.Equal(t, "self", string(model.DirectionSelf))
}

func TestSetOperationDirection_SelfSend(t *testing.T) {
	const viewer = "GVIEWER"
	op := model.Operation{Type: "payment", From: viewer, To: viewer}
	setOperationDirection(&op, viewer)
	assert.Equal(t, model.DirectionSelf, op.Direction)
	assert.Equal(t, "", op.Counterparty, "self-payments have no distinct counterparty")
}

// --- fetchTransactionGroups state machine tests -------------------------------------------

// fakeFetcher is a scripted operationFetcher that returns pre-built pages in order and records
// the requests it received. Used to drive the grouping state machine without a live Horizon.
type fakeFetcher struct {
	pages    []operations.OperationsPage
	err      error
	requests []horizonclient.OperationRequest
	calls    int
}

func (f *fakeFetcher) Operations(req horizonclient.OperationRequest) (operations.OperationsPage, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return operations.OperationsPage{}, f.err
	}
	if f.calls >= len(f.pages) {
		return operations.OperationsPage{}, nil
	}
	p := f.pages[f.calls]
	f.calls++
	return p, nil
}

// buildPage constructs an OperationsPage from payment ops. Each op gets a synthetic paging
// token ("pt-<index>") and is associated with the given transaction hash so adjacent ops with
// the same hash form a group.
func buildPage(ops ...operations.Payment) operations.OperationsPage {
	page := operations.OperationsPage{}
	for i := range ops {
		ops[i].Base.PT = ops[i].ID
		page.Embedded.Records = append(page.Embedded.Records, ops[i])
	}
	return page
}

// newPayment builds a Horizon Payment op with a real transaction join attached. If memo is
// non-empty, MemoType is set to "text"; otherwise "none".
func newPayment(id, txHash, from, to, amount, memo string) operations.Payment {
	p := operations.Payment{
		Base: operations.Base{
			ID:              id,
			PT:              id,
			Type:            "payment",
			TransactionHash: txHash,
			Transaction: &horizon.Transaction{
				Hash:       txHash,
				Successful: true,
				Account:    from,
				Memo:       memo,
				MemoType:   "none",
			},
		},
		From:   from,
		To:     to,
		Amount: amount,
	}
	p.Asset.Type = "credit_alphanum4"
	p.Asset.Code = "EURMTL"
	if memo != "" {
		p.Base.Transaction.MemoType = "text"
	}
	return p
}

// padPage fills a page to exactly fetchSize records so hasMoreInBlockchain evaluates to true.
// The filler ops all share one filler transaction hash so they do not produce extra groups.
func padPage(page operations.OperationsPage, targetSize int, fillerHash string) operations.OperationsPage {
	for len(page.Embedded.Records) < targetSize {
		idx := len(page.Embedded.Records)
		op := newPayment(
			fmt.Sprintf("filler-%d", idx),
			fillerHash,
			"GFILLER", "GOTHER",
			"10", "",
		)
		page.Embedded.Records = append(page.Embedded.Records, op)
	}
	return page
}

func TestFetchTransactionGroups_EmptyStream(t *testing.T) {
	fetcher := &fakeFetcher{pages: []operations.OperationsPage{{}}}
	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got.Groups)
	assert.False(t, got.HasMore)
	assert.Equal(t, "", got.NextCursor)
}

func TestFetchTransactionGroups_SingleGroupOneOp(t *testing.T) {
	fetcher := &fakeFetcher{
		pages: []operations.OperationsPage{
			buildPage(newPayment("pt-1", "txA", "GVIEWER", "GOTHER", "100", "hi")),
		},
	}
	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.Len(t, got.Groups, 1)
	assert.Equal(t, "txA", got.Groups[0].Hash)
	assert.Len(t, got.Groups[0].Operations, 1)
	assert.Equal(t, model.DirectionOut, got.Groups[0].Operations[0].Direction)
	assert.Equal(t, "GOTHER", got.Groups[0].Operations[0].Counterparty)
	assert.Equal(t, "hi", got.Groups[0].Memo)
	assert.Equal(t, "text", got.Groups[0].MemoType)
	assert.False(t, got.Groups[0].Partial)
	assert.False(t, got.HasMore, "stream exhausted on a partial page")
}

func TestFetchTransactionGroups_MultiOpGroup(t *testing.T) {
	fetcher := &fakeFetcher{
		pages: []operations.OperationsPage{
			buildPage(
				newPayment("pt-1", "txA", "GVIEWER", "GOTHER", "100", ""),
				newPayment("pt-2", "txA", "GVIEWER", "GOTHER2", "50", ""),
				newPayment("pt-3", "txA", "GVIEWER", "GOTHER3", "25", ""),
			),
		},
	}
	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.Len(t, got.Groups, 1)
	assert.Len(t, got.Groups[0].Operations, 3, "all three ops should be in the same group")
}

func TestFetchTransactionGroups_GroupLimitBoundary(t *testing.T) {
	// 12 distinct transactions in one page; request limit 10. Should return 10 groups and
	// report HasMore=true even though the remaining records were in the same partial page
	// (this is the C1 regression: previously HasMore was false, losing the remainder).
	var ops []operations.Payment
	for i := 0; i < 12; i++ {
		ops = append(ops, newPayment(
			fmt.Sprintf("pt-%d", i),
			fmt.Sprintf("tx-%d", i),
			"GVIEWER", "GOTHER", "100", "",
		))
	}
	fetcher := &fakeFetcher{pages: []operations.OperationsPage{buildPage(ops...)}}

	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.Len(t, got.Groups, 10)
	assert.True(t, got.HasMore, "partial page with unread records must advertise HasMore")
	assert.NotEmpty(t, got.NextCursor)
	// Cursor should point to the 10th op (pt-9) — the last op of the last completed group.
	assert.Equal(t, "pt-9", got.NextCursor)
}

func TestFetchTransactionGroups_BlockchainExhausted(t *testing.T) {
	// Three transactions spread across one partial page. Since the page is not saturated,
	// blockchain is exhausted → the in-progress group must be flushed.
	fetcher := &fakeFetcher{
		pages: []operations.OperationsPage{
			buildPage(
				newPayment("pt-1", "txA", "GVIEWER", "GOTHER", "100", ""),
				newPayment("pt-2", "txB", "GVIEWER", "GOTHER", "50", ""),
				newPayment("pt-3", "txB", "GVIEWER", "GOTHER2", "25", ""),
			),
		},
	}
	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.Len(t, got.Groups, 2)
	assert.Equal(t, "txA", got.Groups[0].Hash)
	assert.Equal(t, "txB", got.Groups[1].Hash)
	assert.Len(t, got.Groups[1].Operations, 2)
	assert.False(t, got.HasMore)
	assert.Equal(t, "pt-3", got.NextCursor)
}

func TestFetchTransactionGroups_AllSpamGroupDropped(t *testing.T) {
	// Transaction where all ops are sub-1-XLM native payments — should be dropped entirely.
	spamOp := newPayment("pt-1", "txSpam", "GOTHER", "GVIEWER", "0.5", "")
	spamOp.Asset.Type = "native"
	spamOp.Asset.Code = ""

	realOp := newPayment("pt-2", "txReal", "GOTHER", "GVIEWER", "100", "")

	fetcher := &fakeFetcher{
		pages: []operations.OperationsPage{buildPage(spamOp, realOp)},
	}
	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.Len(t, got.Groups, 1, "all-spam group should be dropped, real group kept")
	assert.Equal(t, "txReal", got.Groups[0].Hash)
}

func TestFetchTransactionGroups_MixedSpamKeptInGroup(t *testing.T) {
	// Transaction with one spam op and one real op — whole group preserved (transaction-level
	// filtering, not op-level). This is the H1 fix.
	spamOp := newPayment("pt-1", "txMixed", "GOTHER", "GVIEWER", "0.5", "")
	spamOp.Asset.Type = "native"
	spamOp.Asset.Code = ""

	realOp := newPayment("pt-2", "txMixed", "GOTHER", "GVIEWER", "100", "")

	fetcher := &fakeFetcher{
		pages: []operations.OperationsPage{buildPage(spamOp, realOp)},
	}
	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.Len(t, got.Groups, 1)
	assert.Len(t, got.Groups[0].Operations, 2, "mixed group keeps both spam and real ops")
}

func TestFetchTransactionGroups_ForceFlushOnCap(t *testing.T) {
	// Simulate maxUnderlyingOps (200) ops all in one transaction: filler ops on the same hash.
	// Each page is fetchSize-saturated so hasMoreInBlockchain stays true.
	var pages []operations.OperationsPage
	for i := 0; i < maxIterations; i++ {
		var ops []operations.Payment
		for j := 0; j < fetchSize; j++ {
			idx := i*fetchSize + j
			ops = append(ops, newPayment(
				fmt.Sprintf("pt-%d", idx),
				"txHuge",
				"GVIEWER", "GOTHER", "100", "",
			))
		}
		pages = append(pages, buildPage(ops...))
	}
	fetcher := &fakeFetcher{pages: pages}

	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.Len(t, got.Groups, 1, "force-flush should emit the partial group")
	assert.True(t, got.Groups[0].Partial, "force-flushed group must be marked Partial")
	assert.True(t, got.HasMore)
	assert.NotEmpty(t, got.NextCursor, "cursor must advance past the partial group")
}

func TestFetchTransactionGroups_AllSpamWindowDeadlockGuard(t *testing.T) {
	// Construct pages that are each fully saturated with spam ops under the same tx hash.
	// The grouping loop will buffer them all, then the force-flush fires because nothing
	// completed — but groupHasAnyRealOp drops the group as all-spam. Result: no completed
	// groups, no advanced cursor. The defensive guard must set HasMore=false.
	var pages []operations.OperationsPage
	for i := 0; i < maxIterations; i++ {
		var ops []operations.Payment
		for j := 0; j < fetchSize; j++ {
			idx := i*fetchSize + j
			spam := newPayment(
				fmt.Sprintf("pt-%d", idx),
				"txAllSpam",
				"GOTHER", "GVIEWER", "0.5", "",
			)
			spam.Asset.Type = "native"
			spam.Asset.Code = ""
			ops = append(ops, spam)
		}
		pages = append(pages, buildPage(ops...))
	}
	fetcher := &fakeFetcher{pages: pages}

	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.Empty(t, got.Groups)
	assert.False(t, got.HasMore, "guard must flip HasMore to false when cursor is empty")
	assert.Equal(t, "", got.NextCursor)
}

func TestFetchTransactionGroups_HorizonError(t *testing.T) {
	fetcher := &fakeFetcher{err: errors.New("horizon unreachable")}
	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestFetchTransactionGroups_CursorPassedThrough(t *testing.T) {
	fetcher := &fakeFetcher{pages: []operations.OperationsPage{{}}}
	_, err := fetchTransactionGroups(fetcher, "GVIEWER", "abc123", 10)
	assert.NoError(t, err)
	assert.Len(t, fetcher.requests, 1)
	assert.Equal(t, "abc123", fetcher.requests[0].Cursor)
	assert.Equal(t, "transactions", fetcher.requests[0].Join)
	assert.Equal(t, horizonclient.OrderDesc, fetcher.requests[0].Order)
}

func TestFetchTransactionGroups_LedgerCloseTimeUsedForCreatedAt(t *testing.T) {
	closeTime := time.Date(2026, 4, 9, 12, 30, 45, 0, time.UTC)
	op := newPayment("pt-1", "txA", "GVIEWER", "GOTHER", "100", "")
	op.Base.Transaction.LedgerCloseTime = closeTime
	fetcher := &fakeFetcher{pages: []operations.OperationsPage{buildPage(op)}}

	got, err := fetchTransactionGroups(fetcher, "GVIEWER", "", 10)
	assert.NoError(t, err)
	assert.Len(t, got.Groups, 1)
	assert.Equal(t, "2026-04-09 12:30:45", got.Groups[0].CreatedAt, "group CreatedAt should come from LedgerCloseTime, not op timestamp")
}
