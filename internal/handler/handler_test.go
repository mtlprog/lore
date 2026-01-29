package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtlprog/lore/internal/config"
	"github.com/mtlprog/lore/internal/handler/mocks"
	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/repository"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Constructor tests

func TestNewHandler(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountQuerier(t)
	tmpl := mocks.NewMockTemplateRenderer(t)

	t.Run("nil stellar service returns error", func(t *testing.T) {
		h, err := New(nil, accounts, tmpl)
		assert.Nil(t, h)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stellar service")
	})

	t.Run("nil account repository returns error", func(t *testing.T) {
		h, err := New(stellar, nil, tmpl)
		assert.Nil(t, h)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account repository")
	})

	t.Run("nil templates returns error", func(t *testing.T) {
		h, err := New(stellar, accounts, nil)
		assert.Nil(t, h)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "templates")
	})

	t.Run("valid dependencies returns handler", func(t *testing.T) {
		h, err := New(stellar, accounts, tmpl)
		assert.NoError(t, err)
		assert.NotNil(t, h)
	})
}

// Home handler tests

func TestHomeHandler(t *testing.T) {
	t.Run("successful render with data", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{
			TotalAccounts:  100,
			TotalPersons:   50,
			TotalCompanies: 25,
			TotalXLMValue:  1000000.0,
		}, nil)

		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return([]repository.PersonRow{
			{AccountID: "GABC", Name: "Test Person", MTLAPBalance: 100.0},
		}, nil)

		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, mock.Anything).Return([]repository.CorporateRow{
			{AccountID: "GDEF", Name: "Test Company", MTLACBalance: 50.0, TotalXLMValue: 5000.0},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "home.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		homeData, ok := renderedData.(HomeData)
		require.True(t, ok)
		assert.Equal(t, 100, homeData.Stats.TotalAccounts)
		assert.Len(t, homeData.Persons, 1)
		assert.Len(t, homeData.Corporate, 1)
	})

	t.Run("pagination parameters parsed correctly", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		// Expect offset 20 for persons and 40 for corporate
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, 20).Return(nil, nil)
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, 40).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/?persons_offset=20&corporate_offset=40", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		// Test passes if the expectations were met (correct offsets passed)
	})

	t.Run("negative offset defaults to zero", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, 0).Return(nil, nil)
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/?persons_offset=-5", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		// Test passes if GetPersons was called with offset 0
	})

	t.Run("invalid offset defaults to zero", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, 0).Return(nil, nil)
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/?persons_offset=abc", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		// Test passes if GetPersons was called with offset 0
	})

	t.Run("stats error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(nil, errors.New("database error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch stats")
	})

	t.Run("persons error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch persons")
	})

	t.Run("corporate error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch corporate")
	})

	t.Run("template render error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("template error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("has more pagination flags set correctly", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		// Return 21 items (more than DefaultPageLimit of 20)
		persons := make([]repository.PersonRow, 21)
		for i := range persons {
			persons[i] = repository.PersonRow{AccountID: "G" + string(rune('A'+i))}
		}

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(persons, nil)
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		homeData := renderedData.(HomeData)
		assert.True(t, homeData.HasMorePersons)
		assert.Len(t, homeData.Persons, config.DefaultPageLimit) // Should be truncated to DefaultPageLimit
	})
}

// Account handler tests

func TestAccountHandler(t *testing.T) {
	t.Run("missing account ID returns 400", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		// Create request with empty path value
		req := httptest.NewRequest(http.MethodGet, "/accounts/", nil)
		req.SetPathValue("id", "")
		w := httptest.NewRecorder()

		h.Account(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Account ID required")
	})

	t.Run("successful account fetch renders template", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetAccountDetail(mock.Anything, "GABC123").Return(&model.AccountDetail{
			ID:       "GABC123",
			Name:     "Test Account",
			About:    "Test description",
			Websites: []string{"https://example.com"},
			Trustlines: []model.Trustline{
				{AssetCode: "XLM", Balance: "100"},
			},
		}, nil)

		// Expect relationship/trust rating calls
		accounts.EXPECT().GetRelationships(mock.Anything, "GABC123").Return(nil, nil)
		accounts.EXPECT().GetTrustRatings(mock.Anything, "GABC123").Return(&repository.TrustRating{}, nil)
		accounts.EXPECT().GetConfirmedRelationships(mock.Anything, "GABC123").Return(nil, nil)
		accounts.EXPECT().GetAccountInfo(mock.Anything, "GABC123").Return(&repository.AccountInfo{}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "account.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123", nil)
		req.SetPathValue("id", "GABC123")
		w := httptest.NewRecorder()

		h.Account(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		accountData, ok := renderedData.(AccountData)
		require.True(t, ok)
		assert.Equal(t, "GABC123", accountData.Account.ID)
		assert.Equal(t, "Test Account", accountData.Account.Name)
	})

	t.Run("stellar service error returns 500", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetAccountDetail(mock.Anything, "GABC123").Return(nil, errors.New("horizon error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123", nil)
		req.SetPathValue("id", "GABC123")
		w := httptest.NewRecorder()

		h.Account(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch account")
	})

	t.Run("template render error returns 500", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetAccountDetail(mock.Anything, "GABC123").Return(&model.AccountDetail{ID: "GABC123"}, nil)
		accounts.EXPECT().GetRelationships(mock.Anything, "GABC123").Return(nil, nil)
		accounts.EXPECT().GetTrustRatings(mock.Anything, "GABC123").Return(&repository.TrustRating{}, nil)
		accounts.EXPECT().GetConfirmedRelationships(mock.Anything, "GABC123").Return(nil, nil)
		accounts.EXPECT().GetAccountInfo(mock.Anything, "GABC123").Return(&repository.AccountInfo{}, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("template error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123", nil)
		req.SetPathValue("id", "GABC123")
		w := httptest.NewRecorder()

		h.Account(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("account not found returns 404", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		// Create a Horizon "not found" error
		notFoundErr := &horizonclient.Error{
			Response: &http.Response{StatusCode: 404},
		}
		stellar.EXPECT().GetAccountDetail(mock.Anything, "GNOTFOUND").Return(nil, notFoundErr)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/accounts/GNOTFOUND", nil)
		req.SetPathValue("id", "GNOTFOUND")
		w := httptest.NewRecorder()

		h.Account(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Account not found")
	})
}

// RegisterRoutes test

func TestRegisterRoutes(t *testing.T) {
	t.Run("home route registered", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		// Set up expectations for home route
		accounts.EXPECT().GetStats(mock.Anything).Return(nil, errors.New("expected error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return 500 because GetStats fails, not 404 (route is registered)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("account route registered", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		// Set up expectations for account route
		stellar.EXPECT().GetAccountDetail(mock.Anything, "test").Return(nil, errors.New("expected error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/accounts/test", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return 500 because GetAccountDetail fails, not 404 (route is registered)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("POST method not allowed", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		// POST to a GET-only route should fail
		req := httptest.NewRequest(http.MethodPost, "/accounts/test", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Go 1.22+ returns 405 Method Not Allowed for wrong method
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

// groupRelationships tests

func TestGroupRelationships(t *testing.T) {
	const accountID = "GTEST1234567890123456789012345678901234567890123456"

	t.Run("empty rows returns empty categories", func(t *testing.T) {
		categories := groupRelationships(accountID, nil, nil)

		assert.Len(t, categories, 5) // All category definitions
		for _, cat := range categories {
			assert.True(t, cat.IsEmpty)
			assert.Empty(t, cat.Relationships)
		}
	})

	t.Run("complementary pair MyPart/PartOf shows as confirmed", func(t *testing.T) {
		rows := []repository.RelationshipRow{
			{
				SourceAccountID: accountID,
				TargetAccountID: "GORG12345678901234567890123456789012345678901234567",
				TargetName:      "Test Org",
				RelationType:    "PartOf",
				RelationIndex:   "",
				Direction:       "outgoing",
			},
			{
				SourceAccountID: "GORG12345678901234567890123456789012345678901234567",
				TargetAccountID: accountID,
				TargetName:      "Test Org",
				RelationType:    "MyPart",
				RelationIndex:   "",
				Direction:       "incoming",
			},
		}

		categories := groupRelationships(accountID, rows, nil)

		// Find NETWORK category
		var networkCat *model.RelationshipCategory
		for i := range categories {
			if categories[i].Name == "NETWORK" {
				networkCat = &categories[i]
				break
			}
		}

		require.NotNil(t, networkCat)
		assert.False(t, networkCat.IsEmpty)
		// Should have 1 merged relationship (not 2 separate)
		assert.Len(t, networkCat.Relationships, 1)
		assert.True(t, networkCat.Relationships[0].IsConfirmed)
	})

	t.Run("symmetric type with one-way declaration is hidden", func(t *testing.T) {
		rows := []repository.RelationshipRow{
			{
				SourceAccountID: accountID,
				TargetAccountID: "GOTHER12345678901234567890123456789012345678901234",
				TargetName:      "Other Account",
				RelationType:    "FactionMember",
				RelationIndex:   "",
				Direction:       "outgoing",
			},
		}

		categories := groupRelationships(accountID, rows, nil)

		// Find SOCIAL category
		var socialCat *model.RelationshipCategory
		for i := range categories {
			if categories[i].Name == "SOCIAL" {
				socialCat = &categories[i]
				break
			}
		}

		require.NotNil(t, socialCat)
		// One-way FactionMember should be hidden
		assert.True(t, socialCat.IsEmpty)
		assert.Empty(t, socialCat.Relationships)
	})

	t.Run("symmetric type with mutual declaration is shown", func(t *testing.T) {
		otherID := "GOTHER12345678901234567890123456789012345678901234"
		rows := []repository.RelationshipRow{
			{
				SourceAccountID: accountID,
				TargetAccountID: otherID,
				TargetName:      "Other Account",
				RelationType:    "FactionMember",
				RelationIndex:   "",
				Direction:       "outgoing",
			},
			{
				SourceAccountID: otherID,
				TargetAccountID: accountID,
				TargetName:      "Other Account",
				RelationType:    "FactionMember",
				RelationIndex:   "",
				Direction:       "incoming",
			},
		}

		categories := groupRelationships(accountID, rows, nil)

		// Find SOCIAL category
		var socialCat *model.RelationshipCategory
		for i := range categories {
			if categories[i].Name == "SOCIAL" {
				socialCat = &categories[i]
				break
			}
		}

		require.NotNil(t, socialCat)
		assert.False(t, socialCat.IsEmpty)
		// Should have exactly 1 relationship (deduplicated)
		assert.Len(t, socialCat.Relationships, 1)
		assert.True(t, socialCat.Relationships[0].IsMutual)
	})

	t.Run("mutual relationships are deduplicated", func(t *testing.T) {
		otherID := "GOTHER12345678901234567890123456789012345678901234"
		rows := []repository.RelationshipRow{
			{
				SourceAccountID: accountID,
				TargetAccountID: otherID,
				TargetName:      "Other Account",
				RelationType:    "Partnership",
				RelationIndex:   "",
				Direction:       "outgoing",
			},
			{
				SourceAccountID: otherID,
				TargetAccountID: accountID,
				TargetName:      "Other Account",
				RelationType:    "Partnership",
				RelationIndex:   "",
				Direction:       "incoming",
			},
		}

		categories := groupRelationships(accountID, rows, nil)

		// Find NETWORK category (Partnership is in NETWORK)
		var networkCat *model.RelationshipCategory
		for i := range categories {
			if categories[i].Name == "NETWORK" {
				networkCat = &categories[i]
				break
			}
		}

		require.NotNil(t, networkCat)
		// Should have exactly 1 relationship, not 2
		assert.Len(t, networkCat.Relationships, 1)
		assert.True(t, networkCat.Relationships[0].IsMutual)
	})

	t.Run("unknown relationship type is skipped", func(t *testing.T) {
		rows := []repository.RelationshipRow{
			{
				SourceAccountID: accountID,
				TargetAccountID: "GOTHER12345678901234567890123456789012345678901234",
				TargetName:      "Other Account",
				RelationType:    "UnknownType",
				RelationIndex:   "",
				Direction:       "outgoing",
			},
		}

		categories := groupRelationships(accountID, rows, nil)

		// All categories should be empty
		for _, cat := range categories {
			assert.True(t, cat.IsEmpty)
		}
	})

	t.Run("relationships grouped into correct categories", func(t *testing.T) {
		rows := []repository.RelationshipRow{
			{
				SourceAccountID: accountID,
				TargetAccountID: "GFAMILY123456789012345678901234567890123456789012",
				TargetName:      "Family Member",
				RelationType:    "Sympathy",
				RelationIndex:   "",
				Direction:       "outgoing",
			},
			{
				SourceAccountID: accountID,
				TargetAccountID: "GWORK12345678901234567890123456789012345678901234",
				TargetName:      "Employer",
				RelationType:    "Contractor",
				RelationIndex:   "",
				Direction:       "outgoing",
			},
			{
				SourceAccountID: accountID,
				TargetAccountID: "GOWNER12345678901234567890123456789012345678901234",
				TargetName:      "My Company",
				RelationType:    "Owner",
				RelationIndex:   "",
				Direction:       "outgoing",
			},
		}

		categories := groupRelationships(accountID, rows, nil)

		// Check FAMILY category
		var familyCat, workCat, ownershipCat *model.RelationshipCategory
		for i := range categories {
			switch categories[i].Name {
			case "FAMILY":
				familyCat = &categories[i]
			case "WORK":
				workCat = &categories[i]
			case "OWNERSHIP":
				ownershipCat = &categories[i]
			}
		}

		require.NotNil(t, familyCat)
		require.NotNil(t, workCat)
		require.NotNil(t, ownershipCat)

		assert.Len(t, familyCat.Relationships, 1)
		assert.Equal(t, "Sympathy", familyCat.Relationships[0].Type)

		assert.Len(t, workCat.Relationships, 1)
		assert.Equal(t, "Contractor", workCat.Relationships[0].Type)

		assert.Len(t, ownershipCat.Relationships, 1)
		assert.Equal(t, "Owner", ownershipCat.Relationships[0].Type)
	})
}

// calculateTrustGrade tests

func TestCalculateTrustGrade(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{"perfect score", 4.0, "A"},
		{"high A", 3.5, "A"},
		{"boundary A-", 3.49, "A-"},
		{"A-", 3.0, "A-"},
		{"boundary B+", 2.99, "B+"},
		{"B+", 2.5, "B+"},
		{"B", 2.0, "B"},
		{"C+", 1.5, "C+"},
		{"C", 1.0, "C"},
		{"D high", 0.5, "D"},
		{"D zero", 0.0, "D"},
		{"D negative", -1.0, "D"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTrustGrade(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}
