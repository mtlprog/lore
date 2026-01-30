package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtlprog/lore/internal/handler/mocks"
	"github.com/mtlprog/lore/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTokenHandler(t *testing.T) {
	validIssuer := "GABCDEFGHIJKLMNOPQRSTUVWXYZ234567890ABCDEFGHIJKLMNOPQRST" // 56 chars, starts with G

	t.Run("missing asset code returns 400", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/tokens/"+validIssuer+"/", nil)
		req.SetPathValue("issuer", validIssuer)
		req.SetPathValue("code", "")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Asset code required")
	})

	t.Run("invalid issuer format returns 400", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		// Too short issuer
		req := httptest.NewRequest(http.MethodGet, "/tokens/GABC/MTLAP", nil)
		req.SetPathValue("issuer", "GABC")
		req.SetPathValue("code", "MTLAP")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid issuer address")
	})

	t.Run("issuer with wrong prefix returns 400", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		// S prefix is for secret keys
		invalidIssuer := "SABCDEFGHIJKLMNOPQRSTUVWXYZ234567890ABCDEFGHIJKLMNOPQRST"
		req := httptest.NewRequest(http.MethodGet, "/tokens/"+invalidIssuer+"/MTLAP", nil)
		req.SetPathValue("issuer", invalidIssuer)
		req.SetPathValue("code", "MTLAP")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid issuer address")
	})

	t.Run("token not found returns 404", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetTokenDetail(mock.Anything, "MTLAP", validIssuer).Return(nil, errors.New("token not found"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/tokens/"+validIssuer+"/MTLAP", nil)
		req.SetPathValue("issuer", validIssuer)
		req.SetPathValue("code", "MTLAP")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Token not found")
	})

	t.Run("stellar service error returns 500", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetTokenDetail(mock.Anything, "MTLAP", validIssuer).Return(nil, errors.New("horizon error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/tokens/"+validIssuer+"/MTLAP", nil)
		req.SetPathValue("issuer", validIssuer)
		req.SetPathValue("code", "MTLAP")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch token")
	})

	t.Run("successful token fetch renders template", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetTokenDetail(mock.Anything, "MTLAP", validIssuer).Return(&model.TokenDetail{
			AssetCode:   "MTLAP",
			AssetIssuer: validIssuer,
			IssuerName:  "MTLA Foundation",
			NumAccounts: 100,
			Amount:      "1000000.0000000",
			HomeDomain:  "mtla.me",
		}, nil)

		stellar.EXPECT().GetTokenOrderbook(mock.Anything, "MTLAP", validIssuer, 5).Return(&model.TokenOrderbook{
			Bids: []model.OrderbookEntry{{Price: "10.5", Amount: "100"}},
			Asks: []model.OrderbookEntry{{Price: "11.0", Amount: "50"}},
		}, nil)

		stellar.EXPECT().FetchStellarToml(mock.Anything, "mtla.me").Return(nil, "", nil)
		stellar.EXPECT().GetIssuerNFTMetadata(mock.Anything, validIssuer, "MTLAP").Return(nil, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "token.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/tokens/"+validIssuer+"/MTLAP", nil)
		req.SetPathValue("issuer", validIssuer)
		req.SetPathValue("code", "MTLAP")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		tokenData, ok := renderedData.(TokenData)
		require.True(t, ok)
		assert.Equal(t, "MTLAP", tokenData.Token.AssetCode)
		assert.Equal(t, validIssuer, tokenData.Token.AssetIssuer)
		assert.Equal(t, "MTLA Foundation", tokenData.Token.IssuerName)
		assert.Equal(t, "10.5", tokenData.Token.BestBid)
		assert.Equal(t, "11.0", tokenData.Token.BestAsk)
		assert.NotNil(t, tokenData.Orderbook)
	})

	t.Run("orderbook error continues without market data", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetTokenDetail(mock.Anything, "MTLAP", validIssuer).Return(&model.TokenDetail{
			AssetCode:   "MTLAP",
			AssetIssuer: validIssuer,
			HomeDomain:  "", // No home domain, so FetchStellarToml won't be called
		}, nil)

		stellar.EXPECT().GetTokenOrderbook(mock.Anything, "MTLAP", validIssuer, 5).Return(nil, errors.New("orderbook error"))
		// FetchStellarToml is NOT called when HomeDomain is empty
		stellar.EXPECT().GetIssuerNFTMetadata(mock.Anything, validIssuer, "MTLAP").Return(nil, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "token.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/tokens/"+validIssuer+"/MTLAP", nil)
		req.SetPathValue("issuer", validIssuer)
		req.SetPathValue("code", "MTLAP")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		tokenData := renderedData.(TokenData)
		assert.Nil(t, tokenData.Orderbook)
		assert.Empty(t, tokenData.Token.BestBid)
		assert.Empty(t, tokenData.Token.BestAsk)
	})

	t.Run("NFT metadata is populated when available", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetTokenDetail(mock.Anything, "NFTTEST", validIssuer).Return(&model.TokenDetail{
			AssetCode:   "NFTTEST",
			AssetIssuer: validIssuer,
			HomeDomain:  "", // No home domain
		}, nil)

		stellar.EXPECT().GetTokenOrderbook(mock.Anything, "NFTTEST", validIssuer, 5).Return(nil, nil)
		// FetchStellarToml is NOT called when HomeDomain is empty
		stellar.EXPECT().GetIssuerNFTMetadata(mock.Anything, validIssuer, "NFTTEST").Return(&model.NFTMetadata{
			Name:        "Test NFT",
			Description: "A test NFT",
			ImageURL:    "https://ipfs.io/ipfs/abc123",
		}, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "token.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/tokens/"+validIssuer+"/NFTTEST", nil)
		req.SetPathValue("issuer", validIssuer)
		req.SetPathValue("code", "NFTTEST")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		tokenData := renderedData.(TokenData)
		assert.True(t, tokenData.Token.IsNFT)
		assert.NotNil(t, tokenData.Token.NFTMetadata)
		assert.Equal(t, "Test NFT", tokenData.Token.NFTMetadata.Name)
		assert.Equal(t, "https://ipfs.io/ipfs/abc123", tokenData.Token.ImageURL)
	})

	t.Run("stellar.toml info populates description and image", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetTokenDetail(mock.Anything, "MTLAP", validIssuer).Return(&model.TokenDetail{
			AssetCode:   "MTLAP",
			AssetIssuer: validIssuer,
			HomeDomain:  "mtla.me",
		}, nil)

		stellar.EXPECT().GetTokenOrderbook(mock.Anything, "MTLAP", validIssuer, 5).Return(nil, nil)
		stellar.EXPECT().FetchStellarToml(mock.Anything, "mtla.me").Return(nil, `[[CURRENCIES]]
code = "MTLAP"
issuer = "`+validIssuer+`"
desc = "Montelibero Person Token"
image = "https://mtla.me/logo.png"
`, nil)
		stellar.EXPECT().GetIssuerNFTMetadata(mock.Anything, validIssuer, "MTLAP").Return(nil, nil)

		var renderedData any
		tmpl.EXPECT().Render(mock.Anything, "token.html", mock.Anything).Run(func(w io.Writer, name string, data any) {
			renderedData = data
		}).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/tokens/"+validIssuer+"/MTLAP", nil)
		req.SetPathValue("issuer", validIssuer)
		req.SetPathValue("code", "MTLAP")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		tokenData := renderedData.(TokenData)
		assert.Equal(t, "Montelibero Person Token", tokenData.Token.Description)
		assert.Equal(t, "https://mtla.me/logo.png", tokenData.Token.ImageURL)
	})

	t.Run("template render error returns 500", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		stellar.EXPECT().GetTokenDetail(mock.Anything, "MTLAP", validIssuer).Return(&model.TokenDetail{
			AssetCode:   "MTLAP",
			AssetIssuer: validIssuer,
			HomeDomain:  "", // No home domain
		}, nil)

		stellar.EXPECT().GetTokenOrderbook(mock.Anything, "MTLAP", validIssuer, 5).Return(nil, nil)
		// FetchStellarToml is NOT called when HomeDomain is empty
		stellar.EXPECT().GetIssuerNFTMetadata(mock.Anything, validIssuer, "MTLAP").Return(nil, nil)

		tmpl.EXPECT().Render(mock.Anything, "token.html", mock.Anything).Return(errors.New("template error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/tokens/"+validIssuer+"/MTLAP", nil)
		req.SetPathValue("issuer", validIssuer)
		req.SetPathValue("code", "MTLAP")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("M prefix issuer is valid", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		mIssuer := "MABCDEFGHIJKLMNOPQRSTUVWXYZ234567890ABCDEFGHIJKLMNOPQRST"

		stellar.EXPECT().GetTokenDetail(mock.Anything, "TEST", mIssuer).Return(&model.TokenDetail{
			AssetCode:   "TEST",
			AssetIssuer: mIssuer,
			HomeDomain:  "", // No home domain
		}, nil)

		stellar.EXPECT().GetTokenOrderbook(mock.Anything, "TEST", mIssuer, 5).Return(nil, nil)
		// FetchStellarToml is NOT called when HomeDomain is empty
		stellar.EXPECT().GetIssuerNFTMetadata(mock.Anything, mIssuer, "TEST").Return(nil, nil)

		tmpl.EXPECT().Render(mock.Anything, "token.html", mock.Anything).Return(nil)

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/tokens/"+mIssuer+"/TEST", nil)
		req.SetPathValue("issuer", mIssuer)
		req.SetPathValue("code", "TEST")
		w := httptest.NewRecorder()

		h.Token(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTokenRouteRegistration(t *testing.T) {
	t.Run("token route registered with issuer first", func(t *testing.T) {
		stellar := mocks.NewMockStellarServicer(t)
		accounts := mocks.NewMockAccountQuerier(t)
		tmpl := mocks.NewMockTemplateRenderer(t)

		validIssuer := "GABCDEFGHIJKLMNOPQRSTUVWXYZ234567890ABCDEFGHIJKLMNOPQRST"

		stellar.EXPECT().GetTokenDetail(mock.Anything, "MTLAP", validIssuer).Return(nil, errors.New("expected error"))

		h, err := New(stellar, accounts, tmpl)
		require.NoError(t, err)

		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		// Route is /tokens/{issuer}/{code}
		req := httptest.NewRequest(http.MethodGet, "/tokens/"+validIssuer+"/MTLAP", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return 500 because GetTokenDetail fails, not 404 (route is registered)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
