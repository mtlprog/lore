package handler

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"

	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/service"
)

// TokenData holds data for the token detail page template.
type TokenData struct {
	Token     *model.TokenDetail
	Orderbook *model.TokenOrderbook
}

// Token handles the token detail page.
func (h *Handler) Token(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := r.PathValue("code")
	issuer := r.PathValue("issuer")

	// Validate code
	if code == "" {
		http.Error(w, "Asset code required", http.StatusBadRequest)
		return
	}

	// Validate issuer format (56 characters, starts with G)
	if len(issuer) != 56 || (issuer[0] != 'G' && issuer[0] != 'M') {
		http.Error(w, "Invalid issuer address", http.StatusBadRequest)
		return
	}

	// Get token details
	token, err := h.stellar.GetTokenDetail(ctx, code, issuer)
	if err != nil {
		if errors.Is(err, service.ErrTokenNotFound) || service.IsNotFound(err) {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to fetch token", "code", code, "issuer", issuer, "error", err)
		http.Error(w, "Failed to fetch token", http.StatusInternalServerError)
		return
	}

	// Get orderbook (optional - continue even if fails)
	const orderbookLimit = 5
	orderbook, err := h.stellar.GetTokenOrderbook(ctx, code, issuer, orderbookLimit)
	if err != nil {
		slog.Warn("failed to fetch orderbook", "code", code, "issuer", issuer, "error", err)
		orderbook = nil
	}

	// Get best bid/ask prices from orderbook
	if orderbook != nil {
		if len(orderbook.Bids) > 0 {
			token.BestBid = orderbook.Bids[0].Price
		}
		if len(orderbook.Asks) > 0 {
			token.BestAsk = orderbook.Asks[0].Price
		}
	}

	// Try to get stellar.toml info (optional)
	if token.HomeDomain != "" {
		tomlCurrency, tomlContent, err := h.stellar.FetchStellarToml(ctx, token.HomeDomain)
		if err != nil {
			slog.Debug("failed to fetch stellar.toml", "domain", token.HomeDomain, "error", err)
		} else if tomlContent != "" {
			// Try to find the specific currency in the TOML
			currency := service.FindCurrencyInToml(tomlContent, code, issuer)
			if currency != nil {
				token.Description = currency.Description
				if currency.Image != "" {
					token.ImageURL = currency.Image
				}
			} else if tomlCurrency != nil {
				// Fallback to first currency
				token.Description = tomlCurrency.Description
				if tomlCurrency.Image != "" {
					token.ImageURL = tomlCurrency.Image
				}
			}
		}
	}

	// Try to get NFT metadata (optional)
	nftMeta, err := h.stellar.GetIssuerNFTMetadata(ctx, issuer, code)
	if err != nil {
		slog.Debug("failed to fetch NFT metadata", "code", code, "issuer", issuer, "error", err)
	} else if nftMeta != nil {
		token.IsNFT = true
		token.NFTMetadata = nftMeta
		// NFT image takes precedence
		if nftMeta.ImageURL != "" {
			token.ImageURL = nftMeta.ImageURL
		}
		// NFT description supplements token description
		if nftMeta.Description != "" && token.Description == "" {
			token.Description = nftMeta.Description
		}
	}

	data := TokenData{
		Token:     token,
		Orderbook: orderbook,
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "token.html", data); err != nil {
		slog.Error("failed to render token template", "code", code, "issuer", issuer, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "code", code, "issuer", issuer, "error", err)
	}
}
