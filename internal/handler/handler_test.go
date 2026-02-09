package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
		h, err := New(nil, accounts, nil, tmpl)
		assert.Nil(t, h)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stellar service")
	})

	t.Run("nil account repository returns error", func(t *testing.T) {
		h, err := New(stellar, nil, nil, tmpl)
		assert.Nil(t, h)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account repository")
	})

	t.Run("nil templates returns error", func(t *testing.T) {
		h, err := New(stellar, accounts, nil, nil)
		assert.Nil(t, h)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "templates")
	})

	t.Run("valid dependencies returns handler", func(t *testing.T) {
		h, err := New(stellar, accounts, nil, tmpl)
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
			TotalSynthetic: 10,
			TotalXLMValue:  1000000.0,
		}, nil)

		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return([]repository.PersonRow{
			{AccountID: "GABC", Name: "Test Person", MTLAPBalance: 100.0},
		}, nil)

		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, mock.Anything).Return([]repository.CorporateRow{
			{AccountID: "GDEF", Name: "Test Company", MTLACBalance: 50.0, TotalXLMValue: 5000.0},
		}, nil)

		accounts.EXPECT().GetSynthetic(mock.Anything, mock.Anything, mock.Anything).Return([]repository.SyntheticRow{
			{AccountID: "GHIJ", Name: "Test Synthetic", MTLAXBalance: 1.0, ReputationScore: 3.5, ReputationWeight: 10.0},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "home.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
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
		assert.Len(t, homeData.Synthetic, 1)
		assert.Equal(t, "A", homeData.Synthetic[0].ReputationGrade)
		assert.Equal(t, 10.0, homeData.Synthetic[0].ReputationWeight)
	})

	t.Run("pagination parameters parsed correctly", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		// Expect offset 20 for persons, 40 for corporate, 0 for synthetic
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, 20).Return(nil, nil)
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, 40).Return(nil, nil)
		accounts.EXPECT().GetSynthetic(mock.Anything, mock.Anything, 0).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
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
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, 0).Return(nil, nil)
		accounts.EXPECT().GetSynthetic(mock.Anything, mock.Anything, 0).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
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
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, 0).Return(nil, nil)
		accounts.EXPECT().GetSynthetic(mock.Anything, mock.Anything, 0).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
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

		h, err := New(stellar, accounts, nil, tmpl)
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

		h, err := New(stellar, accounts, nil, tmpl)
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

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch corporate")
	})

	t.Run("synthetic error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		accounts.EXPECT().GetSynthetic(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch synthetic")
	})

	t.Run("template render error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		accounts.EXPECT().GetCorporate(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		accounts.EXPECT().GetSynthetic(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("template error"))

		h, err := New(stellar, accounts, nil, tmpl)
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
		accounts.EXPECT().GetSynthetic(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
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

		h, err := New(stellar, accounts, nil, tmpl)
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
		accounts.EXPECT().GetLPShares(mock.Anything, "GABC123").Return(nil, nil)

		// Expect operations fetch
		stellar.EXPECT().GetAccountOperations(mock.Anything, "GABC123", "", 10).Return(&model.OperationsPage{
			Operations: []model.Operation{},
			HasMore:    false,
		}, nil)

		// Expect account names lookup (empty since no operations)
		accounts.EXPECT().GetAccountNames(mock.Anything, mock.Anything).Return(map[string]string{}, nil).Maybe()

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "account.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
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

		h, err := New(stellar, accounts, nil, tmpl)
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
		accounts.EXPECT().GetLPShares(mock.Anything, "GABC123").Return(nil, nil)
		stellar.EXPECT().GetAccountOperations(mock.Anything, "GABC123", "", 10).Return(nil, nil)
		accounts.EXPECT().GetAccountNames(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("template error"))

		h, err := New(stellar, accounts, nil, tmpl)
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

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/accounts/GNOTFOUND", nil)
		req.SetPathValue("id", "GNOTFOUND")
		w := httptest.NewRecorder()

		h.Account(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Account not found")
	})

	t.Run("account with LP shares renders correctly", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetAccountDetail(mock.Anything, "GABC123").Return(&model.AccountDetail{
			ID:         "GABC123",
			Name:       "LP Holder",
			Trustlines: []model.Trustline{},
		}, nil)

		accounts.EXPECT().GetRelationships(mock.Anything, "GABC123").Return(nil, nil)
		accounts.EXPECT().GetTrustRatings(mock.Anything, "GABC123").Return(&repository.TrustRating{}, nil)
		accounts.EXPECT().GetConfirmedRelationships(mock.Anything, "GABC123").Return(nil, nil)
		accounts.EXPECT().GetAccountInfo(mock.Anything, "GABC123").Return(&repository.AccountInfo{}, nil)

		// Return actual LP shares data
		accounts.EXPECT().GetLPShares(mock.Anything, "GABC123").Return([]repository.LPShareRow{
			{
				PoolID:         "abc123poolid",
				ShareBalance:   100.0,
				TotalShares:    1000.0,
				ReserveACode:   "MTL",
				ReserveAIssuer: "GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V",
				ReserveAAmount: 10000.0,
				ReserveBCode:   "XLM",
				ReserveBIssuer: "",
				ReserveBAmount: 50000.0,
				XLMValue:       5000.0,
			},
			{
				PoolID:         "def456poolid",
				ShareBalance:   50.0,
				TotalShares:    500.0,
				ReserveACode:   "EURMTL",
				ReserveAIssuer: "GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V",
				ReserveAAmount: 20000.0,
				ReserveBCode:   "USDC",
				ReserveBIssuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
				ReserveBAmount: 20000.0,
				XLMValue:       2000.0,
			},
		}, nil)

		stellar.EXPECT().GetAccountOperations(mock.Anything, "GABC123", "", 10).Return(&model.OperationsPage{
			Operations: []model.Operation{},
			HasMore:    false,
		}, nil)
		accounts.EXPECT().GetAccountNames(mock.Anything, mock.Anything).Return(map[string]string{}, nil).Maybe()

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "account.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123", nil)
		req.SetPathValue("id", "GABC123")
		w := httptest.NewRecorder()

		h.Account(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		accountData, ok := renderedData.(AccountData)
		require.True(t, ok)

		// Verify LP shares are converted and passed to template
		require.Len(t, accountData.Account.LPShares, 2)

		// Verify first LP share
		lp1 := accountData.Account.LPShares[0]
		assert.Equal(t, "abc123poolid", lp1.PoolID)
		assert.Equal(t, "10.00%", lp1.SharePercent)
		assert.Equal(t, "MTL", lp1.ReserveA.AssetCode)
		assert.Equal(t, "XLM", lp1.ReserveB.AssetCode)
		assert.Equal(t, 5000.0, lp1.XLMValue)

		// Verify second LP share
		lp2 := accountData.Account.LPShares[1]
		assert.Equal(t, "def456poolid", lp2.PoolID)
		assert.Equal(t, "10.00%", lp2.SharePercent)
		assert.Equal(t, "EURMTL", lp2.ReserveA.AssetCode)
		assert.Equal(t, "USDC", lp2.ReserveB.AssetCode)
	})

	t.Run("LP shares fetch error continues without shares", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetAccountDetail(mock.Anything, "GABC123").Return(&model.AccountDetail{
			ID:         "GABC123",
			Trustlines: []model.Trustline{},
		}, nil)

		accounts.EXPECT().GetRelationships(mock.Anything, "GABC123").Return(nil, nil)
		accounts.EXPECT().GetTrustRatings(mock.Anything, "GABC123").Return(&repository.TrustRating{}, nil)
		accounts.EXPECT().GetConfirmedRelationships(mock.Anything, "GABC123").Return(nil, nil)
		accounts.EXPECT().GetAccountInfo(mock.Anything, "GABC123").Return(&repository.AccountInfo{}, nil)

		// LP shares fetch fails
		accounts.EXPECT().GetLPShares(mock.Anything, "GABC123").Return(nil, errors.New("database error"))

		stellar.EXPECT().GetAccountOperations(mock.Anything, "GABC123", "", 10).Return(nil, nil)
		accounts.EXPECT().GetAccountNames(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "account.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123", nil)
		req.SetPathValue("id", "GABC123")
		w := httptest.NewRecorder()

		h.Account(w, req)

		// Page should still render successfully (graceful degradation)
		assert.Equal(t, http.StatusOK, w.Code)

		accountData, ok := renderedData.(AccountData)
		require.True(t, ok)

		// LP shares should be nil/empty when fetch fails
		assert.Nil(t, accountData.Account.LPShares)
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

		h, err := New(stellar, accounts, nil, tmpl)
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

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/accounts/test", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return 500 because GetAccountDetail fails, not 404 (route is registered)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("search route registered", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		// Set up expectations for search route
		accounts.EXPECT().GetAllTags(mock.Anything).Return(nil, errors.New("expected error"))

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return 500 because GetAllTags fails, not 404 (route is registered)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("POST method not allowed", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		h, err := New(stellar, accounts, nil, tmpl)
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

// Transaction handler tests

func TestTransactionHandler(t *testing.T) {
	t.Run("missing transaction hash returns 400", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/transactions/", nil)
		req.SetPathValue("hash", "")
		w := httptest.NewRecorder()

		h.Transaction(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Transaction hash required")
	})

	t.Run("invalid hash format returns 400", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		// Hash too short (not 64 chars)
		req := httptest.NewRequest(http.MethodGet, "/transactions/abc123", nil)
		req.SetPathValue("hash", "abc123")
		w := httptest.NewRecorder()

		h.Transaction(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid transaction hash format")
	})

	t.Run("transaction not found returns 404", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		txHash := "ec8d5d6e64dc4df1bc8d8c200e048d6740d1e9f680612baeda0f78678c9ca666"

		notFoundErr := &horizonclient.Error{
			Response: &http.Response{StatusCode: 404},
		}
		stellar.EXPECT().GetTransactionDetail(mock.Anything, txHash).Return(nil, notFoundErr)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/transactions/"+txHash, nil)
		req.SetPathValue("hash", txHash)
		w := httptest.NewRecorder()

		h.Transaction(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Transaction not found")
	})

	t.Run("stellar service error returns 500", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		txHash := "ec8d5d6e64dc4df1bc8d8c200e048d6740d1e9f680612baeda0f78678c9ca666"

		stellar.EXPECT().GetTransactionDetail(mock.Anything, txHash).Return(nil, errors.New("horizon error"))

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/transactions/"+txHash, nil)
		req.SetPathValue("hash", txHash)
		w := httptest.NewRecorder()

		h.Transaction(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch transaction")
	})

	t.Run("successful transaction fetch renders template", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		txHash := "ec8d5d6e64dc4df1bc8d8c200e048d6740d1e9f680612baeda0f78678c9ca666"

		stellar.EXPECT().GetTransactionDetail(mock.Anything, txHash).Return(&model.Transaction{
			Hash:          txHash,
			Successful:    true,
			SourceAccount: "GABC123",
			Operations: []model.Operation{
				{Type: "payment", From: "GABC123", To: "GDEF456", Amount: "100", AssetCode: "XLM"},
			},
		}, nil)

		accounts.EXPECT().GetAccountNames(mock.Anything, mock.Anything).Return(map[string]string{
			"GABC123": "Alice",
			"GDEF456": "Bob",
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "transaction.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/transactions/"+txHash, nil)
		req.SetPathValue("hash", txHash)
		w := httptest.NewRecorder()

		h.Transaction(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		txData, ok := renderedData.(TransactionData)
		require.True(t, ok)
		assert.Equal(t, txHash, txData.Transaction.Hash)
		assert.True(t, txData.Transaction.Successful)
		assert.Len(t, txData.AccountNames, 2)
	})

	t.Run("template render error returns 500", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		txHash := "ec8d5d6e64dc4df1bc8d8c200e048d6740d1e9f680612baeda0f78678c9ca666"

		stellar.EXPECT().GetTransactionDetail(mock.Anything, txHash).Return(&model.Transaction{
			Hash:          txHash,
			SourceAccount: "GABC123",
			Operations:    []model.Operation{},
		}, nil)
		accounts.EXPECT().GetAccountNames(mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, "transaction.html", mock.Anything).Return(errors.New("template error"))

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/transactions/"+txHash, nil)
		req.SetPathValue("hash", txHash)
		w := httptest.NewRecorder()

		h.Transaction(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("filters claimable balance operations", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		txHash := "ec8d5d6e64dc4df1bc8d8c200e048d6740d1e9f680612baeda0f78678c9ca666"

		stellar.EXPECT().GetTransactionDetail(mock.Anything, txHash).Return(&model.Transaction{
			Hash:          txHash,
			SourceAccount: "GABC123",
			Operations: []model.Operation{
				{Type: "payment", From: "GABC123", To: "GDEF456"},
				{Type: "create_claimable_balance"},
				{Type: "claim_claimable_balance"},
				{Type: "manage_data", DataName: "test"},
			},
			OperationCount: 4,
		}, nil)

		accounts.EXPECT().GetAccountNames(mock.Anything, mock.Anything).Return(nil, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "transaction.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/transactions/"+txHash, nil)
		req.SetPathValue("hash", txHash)
		w := httptest.NewRecorder()

		h.Transaction(w, req)

		txData := renderedData.(TransactionData)
		// Should have filtered out the 2 claimable balance ops, leaving 2
		assert.Len(t, txData.Transaction.Operations, 2)
		assert.Equal(t, 2, txData.Transaction.OperationCount)
	})
}

// collectAccountIDs tests

func TestCollectAccountIDs(t *testing.T) {
	t.Run("empty operations returns empty slice", func(t *testing.T) {
		result := collectAccountIDs(nil)
		assert.Empty(t, result)
	})

	t.Run("collects unique IDs from operations", func(t *testing.T) {
		ops := []model.Operation{
			{From: "GABC", To: "GDEF"},
			{From: "GABC", To: "GHIJ"}, // GABC duplicate
			{SourceAccount: "GKLM"},
		}

		result := collectAccountIDs(ops)

		assert.Len(t, result, 4)
		assert.Contains(t, result, "GABC")
		assert.Contains(t, result, "GDEF")
		assert.Contains(t, result, "GHIJ")
		assert.Contains(t, result, "GKLM")
	})

	t.Run("collects Stellar account IDs from DataValue", func(t *testing.T) {
		// Valid Stellar account ID is exactly 56 characters starting with G or M
		stellarID := "GABCDEFGHIJKLMNOPQRSTUVWXYZ234567890ABCDEFGHIJKLMNOPQRST" // 56 chars
		ops := []model.Operation{
			{DataValue: stellarID},
			{DataValue: "not-a-stellar-id"},
		}

		result := collectAccountIDs(ops)

		assert.Len(t, result, 1)
		assert.Contains(t, result, stellarID)
	})

	t.Run("skips empty fields", func(t *testing.T) {
		ops := []model.Operation{
			{From: "", To: "GDEF", SourceAccount: ""},
		}

		result := collectAccountIDs(ops)

		assert.Len(t, result, 1)
		assert.Contains(t, result, "GDEF")
	})
}

// isStellarAccountID tests

func TestIsStellarAccountID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid G account ID",
			input:    "GABCDEFGHIJKLMNOPQRSTUVWXYZ234567890ABCDEFGHIJKLMNOPQRST", // 56 chars
			expected: true,
		},
		{
			name:     "valid M account ID (muxed)",
			input:    "MABCDEFGHIJKLMNOPQRSTUVWXYZ234567890ABCDEFGHIJKLMNOPQRST", // 56 chars
			expected: true,
		},
		{
			name:     "too short",
			input:    "GABC",
			expected: false,
		},
		{
			name:     "too long (57 chars)",
			input:    "GABCDEFGHIJKLMNOPQRSTUVWXYZ234567890ABCDEFGHIJKLMNOPQRSTU", // 57 chars
			expected: false,
		},
		{
			name:     "wrong prefix (S is for secret key)",
			input:    "SABCDEFGHIJKLMNOPQRSTUVWXYZ234567890ABCDEFGHIJKLMNOPQRST", // 56 chars but S prefix
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "hash-like string (64 chars)",
			input:    "ec8d5d6e64dc4df1bc8d8c200e048d6740d1e9f680612baeda0f78678c9ca666",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStellarAccountID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Search handler tests

func TestSearchHandler(t *testing.T) {
	t.Run("successful render with valid query", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(2, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", mock.Anything, config.DefaultPageLimit+1, 0, mock.Anything).Return([]repository.SearchAccountRow{
			{AccountID: "GABC", Name: "Test Person", MTLAPBalance: 1.0, MTLACBalance: 0, TotalXLMValue: 100},
			{AccountID: "GDEF", Name: "Test Company", MTLAPBalance: 0, MTLACBalance: 1.0, TotalXLMValue: 5000},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData, ok := renderedData.(SearchData)
		require.True(t, ok)
		assert.Equal(t, "test", searchData.Query)
		assert.Len(t, searchData.Accounts, 2)
		assert.Equal(t, 2, searchData.TotalCount)
		assert.True(t, searchData.Accounts[0].IsPerson)
		assert.False(t, searchData.Accounts[0].IsCorporate)
		assert.False(t, searchData.Accounts[1].IsPerson)
		assert.True(t, searchData.Accounts[1].IsCorporate)
	})

	t.Run("empty query renders prompt state with tags cloud", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{
			{TagName: "Belgrade", Count: 10},
			{TagName: "Programmer", Count: 5},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData, ok := renderedData.(SearchData)
		require.True(t, ok)
		assert.Equal(t, "", searchData.Query)
		assert.Empty(t, searchData.Accounts)
		assert.Equal(t, 0, searchData.TotalCount)
		assert.Len(t, searchData.AllTags, 2)
	})

	t.Run("query too short renders prompt state", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=a", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData, ok := renderedData.(SearchData)
		require.True(t, ok)
		assert.Equal(t, "a", searchData.Query)
		assert.Empty(t, searchData.Accounts)
		assert.False(t, searchData.QueryTooLong)
	})

	t.Run("short query with tags still performs search", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{
			{TagName: "Belgrade", Count: 5},
		}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "a", []string{"Belgrade"}).Return(1, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "a", []string{"Belgrade"}, config.DefaultPageLimit+1, 0, mock.Anything).Return([]repository.SearchAccountRow{
			{AccountID: "GTEST1", Name: "Test Account", MTLAPBalance: 10.0},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=a&tag=Belgrade", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData, ok := renderedData.(SearchData)
		require.True(t, ok)
		assert.Equal(t, "a", searchData.Query)
		assert.Equal(t, []string{"Belgrade"}, searchData.Tags)
		assert.Len(t, searchData.Accounts, 1)
		assert.Equal(t, 1, searchData.TotalCount)
	})

	t.Run("query too long sets QueryTooLong flag", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		longQuery := strings.Repeat("a", 101)
		req := httptest.NewRequest(http.MethodGet, "/search?q="+longQuery, nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData, ok := renderedData.(SearchData)
		require.True(t, ok)
		assert.True(t, searchData.QueryTooLong)
		assert.Empty(t, searchData.Accounts)
		assert.Equal(t, 0, searchData.TotalCount)
	})

	t.Run("query too long with tags does not search", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{
			{TagName: "Belgrade", Count: 5},
		}, nil)
		// SearchAccounts should NOT be called because query is too long

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		longQuery := strings.Repeat("a", 101)
		req := httptest.NewRequest(http.MethodGet, "/search?q="+longQuery+"&tag=Belgrade", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData, ok := renderedData.(SearchData)
		require.True(t, ok)
		assert.True(t, searchData.QueryTooLong)
		assert.Empty(t, searchData.Accounts)
		assert.Equal(t, 0, searchData.TotalCount)
		assert.Equal(t, []string{"Belgrade"}, searchData.Tags)
	})

	t.Run("query at exactly 2 chars is valid", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "ab", mock.Anything).Return(0, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "ab", mock.Anything, config.DefaultPageLimit+1, 0, mock.Anything).Return(nil, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=ab", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData, ok := renderedData.(SearchData)
		require.True(t, ok)
		assert.Equal(t, "ab", searchData.Query)
		assert.False(t, searchData.QueryTooLong)
	})

	t.Run("query at exactly 100 chars is valid", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		maxQuery := strings.Repeat("a", 100)
		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, maxQuery, mock.Anything).Return(0, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, maxQuery, mock.Anything, config.DefaultPageLimit+1, 0, mock.Anything).Return(nil, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q="+maxQuery, nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData, ok := renderedData.(SearchData)
		require.True(t, ok)
		assert.False(t, searchData.QueryTooLong)
	})

	t.Run("whitespace trimmed from query", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(0, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", mock.Anything, config.DefaultPageLimit+1, 0, mock.Anything).Return(nil, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=++test++", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		searchData := renderedData.(SearchData)
		assert.Equal(t, "test", searchData.Query)
	})

	t.Run("pagination offset parsed correctly", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(50, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", mock.Anything, config.DefaultPageLimit+1, 20, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=20", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid offset defaults to zero", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(5, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", mock.Anything, config.DefaultPageLimit+1, 0, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=abc", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative offset defaults to zero", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(5, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", mock.Anything, config.DefaultPageLimit+1, 0, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&offset=-5", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("pagination with tags only", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{
			{TagName: "Belgrade", Count: 10},
		}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "", []string{"Belgrade"}).Return(25, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "", []string{"Belgrade"}, config.DefaultPageLimit+1, 20, mock.Anything).Return([]repository.SearchAccountRow{
			{AccountID: "GTEST1", Name: "Test", MTLAPBalance: 10.0},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?tag=Belgrade&offset=20", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData, ok := renderedData.(SearchData)
		require.True(t, ok)
		assert.Equal(t, "", searchData.Query)
		assert.Equal(t, []string{"Belgrade"}, searchData.Tags)
		assert.Equal(t, 20, searchData.Offset)
		assert.Len(t, searchData.Accounts, 1)
	})

	t.Run("HasMore flag set correctly when more results exist", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(30, nil)

		rows := make([]repository.SearchAccountRow, config.DefaultPageLimit+1)
		for i := range rows {
			rows[i] = repository.SearchAccountRow{AccountID: "G" + string(rune('A'+i)), MTLAPBalance: 1.0}
		}
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", mock.Anything, config.DefaultPageLimit+1, 0, mock.Anything).Return(rows, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		searchData := renderedData.(SearchData)
		assert.True(t, searchData.HasMore)
		assert.Len(t, searchData.Accounts, config.DefaultPageLimit)
		assert.Equal(t, config.DefaultPageLimit, searchData.NextOffset)
	})

	t.Run("GetAllTags error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return(nil, errors.New("database error"))

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch tags")
	})

	t.Run("CountSearchAccounts error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(0, errors.New("database error"))

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to search accounts")
	})

	t.Run("SearchAccounts error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(5, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to search accounts")
	})

	t.Run("template render error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(0, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Return(errors.New("template error"))

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("IsPerson and IsCorporate thresholds", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", mock.Anything).Return(4, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]repository.SearchAccountRow{
			{AccountID: "G1", MTLAPBalance: 5.0, MTLACBalance: 0},
			{AccountID: "G2", MTLAPBalance: 5.1, MTLACBalance: 0},
			{AccountID: "G3", MTLAPBalance: 0, MTLACBalance: 4.0},
			{AccountID: "G4", MTLAPBalance: 0, MTLACBalance: 4.1},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		searchData := renderedData.(SearchData)
		assert.True(t, searchData.Accounts[0].IsPerson)
		assert.False(t, searchData.Accounts[1].IsPerson)
		assert.True(t, searchData.Accounts[2].IsCorporate)
		assert.False(t, searchData.Accounts[3].IsCorporate)
	})

	// Tag-specific tests
	t.Run("search by tag only without query", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{
			{TagName: "Belgrade", Count: 10},
		}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "", []string{"Belgrade"}).Return(5, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "", []string{"Belgrade"}, config.DefaultPageLimit+1, 0, mock.Anything).Return([]repository.SearchAccountRow{
			{AccountID: "GABC", Name: "Test Person", MTLAPBalance: 1.0},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?tag=Belgrade", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData := renderedData.(SearchData)
		assert.Equal(t, "", searchData.Query)
		assert.Equal(t, []string{"Belgrade"}, searchData.Tags)
		assert.Len(t, searchData.Accounts, 1)
	})

	t.Run("search with query and tag (AND logic)", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "test", []string{"Belgrade"}).Return(2, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "test", []string{"Belgrade"}, config.DefaultPageLimit+1, 0, mock.Anything).Return([]repository.SearchAccountRow{
			{AccountID: "GABC", Name: "Test Person", MTLAPBalance: 1.0},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?q=test&tag=Belgrade", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData := renderedData.(SearchData)
		assert.Equal(t, "test", searchData.Query)
		assert.Equal(t, []string{"Belgrade"}, searchData.Tags)
	})

	t.Run("search with multiple tags (AND logic)", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "", []string{"Belgrade", "Programmer"}).Return(3, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "", []string{"Belgrade", "Programmer"}, config.DefaultPageLimit+1, 0, mock.Anything).Return([]repository.SearchAccountRow{
			{AccountID: "GABC", Name: "Test Person", MTLAPBalance: 1.0},
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?tag=Belgrade&tag=Programmer", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		searchData := renderedData.(SearchData)
		assert.Equal(t, []string{"Belgrade", "Programmer"}, searchData.Tags)
	})

	t.Run("empty tags are filtered out", func(t *testing.T) {
		accounts := mocks.NewMockAccountQuerier(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		accounts.EXPECT().CountSearchAccounts(mock.Anything, "", []string{"Belgrade"}).Return(0, nil)
		accounts.EXPECT().SearchAccounts(mock.Anything, "", []string{"Belgrade"}, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/search?tag=&tag=Belgrade", nil)
		w := httptest.NewRecorder()

		h.Search(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("search route via mux", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetAllTags(mock.Anything).Return([]repository.TagRow{}, nil)
		tmpl.EXPECT().Render(mock.Anything, "search.html", mock.Anything).Return(nil)

		h, err := New(stellar, accounts, nil, tmpl)
		require.NoError(t, err)

		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// filterValidTags tests

func TestFilterValidTags(t *testing.T) {
	t.Run("filters empty tags", func(t *testing.T) {
		result := filterValidTags([]string{"Belgrade", "", "Programmer"})
		assert.Equal(t, []string{"Belgrade", "Programmer"}, result)
	})

	t.Run("filters overly long tags", func(t *testing.T) {
		longTag := string(make([]byte, 101))
		result := filterValidTags([]string{"Belgrade", longTag, "Programmer"})
		assert.Equal(t, []string{"Belgrade", "Programmer"}, result)
	})

	t.Run("allows max length tags", func(t *testing.T) {
		maxTag := string(make([]byte, 100))
		result := filterValidTags([]string{maxTag})
		assert.Len(t, result, 1)
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		result := filterValidTags([]string{})
		assert.Empty(t, result)
	})

	t.Run("all invalid returns empty slice", func(t *testing.T) {
		result := filterValidTags([]string{"", ""})
		assert.Empty(t, result)
	})
}
