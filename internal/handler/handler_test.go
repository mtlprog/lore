package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtlprog/lore/internal/handler/mocks"
	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Constructor tests

func TestNewHandler(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountRepositoryer(t)
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
		accounts := mocks.NewMockAccountRepositoryer(t)
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

		accounts.EXPECT().GetCompanies(mock.Anything, mock.Anything, mock.Anything).Return([]repository.CompanyRow{
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
		assert.Len(t, homeData.Companies, 1)
	})

	t.Run("pagination parameters parsed correctly", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepositoryer(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		// Expect offset 20 for persons and 40 for companies
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, 20).Return(nil, nil)
		accounts.EXPECT().GetCompanies(mock.Anything, mock.Anything, 40).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/?persons_offset=20&companies_offset=40", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		// Test passes if the expectations were met (correct offsets passed)
	})

	t.Run("negative offset defaults to zero", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepositoryer(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, 0).Return(nil, nil)
		accounts.EXPECT().GetCompanies(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/?persons_offset=-5", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		// Test passes if GetPersons was called with offset 0
	})

	t.Run("invalid offset defaults to zero", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepositoryer(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, 0).Return(nil, nil)
		accounts.EXPECT().GetCompanies(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/?persons_offset=abc", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		// Test passes if GetPersons was called with offset 0
	})

	t.Run("stats error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepositoryer(t)
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
		accounts := mocks.NewMockAccountRepositoryer(t)
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

	t.Run("companies error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepositoryer(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		accounts.EXPECT().GetCompanies(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch companies")
	})

	t.Run("template render error returns 500", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepositoryer(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		accounts.EXPECT().GetCompanies(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("template error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Home(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("has more pagination flags set correctly", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepositoryer(t)
		stellar := mocks.NewMockStellarServicer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		// Return 21 items (more than DefaultPageLimit of 20)
		persons := make([]repository.PersonRow, 21)
		for i := range persons {
			persons[i] = repository.PersonRow{AccountID: "G" + string(rune('A'+i))}
		}

		accounts.EXPECT().GetStats(mock.Anything).Return(&repository.Stats{}, nil)
		accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(persons, nil)
		accounts.EXPECT().GetCompanies(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

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
		assert.Len(t, homeData.Persons, 20) // Should be truncated to DefaultPageLimit
	})
}

// Account handler tests

func TestAccountHandler(t *testing.T) {
	t.Run("missing account ID returns 400", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountRepositoryer(t)
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
		accounts := mocks.NewMockAccountRepositoryer(t)
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
		accounts := mocks.NewMockAccountRepositoryer(t)
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
		accounts := mocks.NewMockAccountRepositoryer(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetAccountDetail(mock.Anything, "GABC123").Return(&model.AccountDetail{ID: "GABC123"}, nil)
		tmpl.EXPECT().Render(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("template error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123", nil)
		req.SetPathValue("id", "GABC123")
		w := httptest.NewRecorder()

		h.Account(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// RegisterRoutes test

func TestRegisterRoutes(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountRepositoryer(t)
	tmpl := mocks.NewMockTemplateRenderer(t)

	// Set up expectations for when routes are called
	accounts.EXPECT().GetStats(mock.Anything).Return(nil, errors.New("not implemented")).Maybe()
	accounts.EXPECT().GetPersons(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("not implemented")).Maybe()
	accounts.EXPECT().GetCompanies(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("not implemented")).Maybe()
	stellar.EXPECT().GetAccountDetail(mock.Anything, mock.Anything).Return(nil, errors.New("not implemented")).Maybe()

	h, err := New(stellar, accounts, tmpl)
	require.NoError(t, err)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Test that routes are registered by checking if they don't 404
	// Note: The actual handler functionality is tested separately

	t.Run("home route registered", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		// Should not be 404 (routes are registered)
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("account route registered", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/accounts/test", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		// Should not be 404 (routes are registered)
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}
