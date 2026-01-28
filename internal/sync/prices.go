package sync

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/stellar/go/clients/horizonclient"
)

// Asset represents a Stellar asset.
type Asset struct {
	Code   string
	Issuer string
}

// syncTokenPrices fetches token prices from SDEX and stores them.
func (s *Syncer) syncTokenPrices(ctx context.Context) error {
	// Get unique assets from account_balances
	assets, err := s.repo.GetUniqueAssets(ctx)
	if err != nil {
		return fmt.Errorf("get unique assets: %w", err)
	}

	slog.Debug("fetching prices for assets", "count", len(assets))

	for _, asset := range assets {
		// Skip native XLM - its price is always 1 XLM
		if asset.Code == "XLM" && asset.Issuer == "" {
			if err := s.repo.UpsertTokenPrice(ctx, asset.Code, asset.Issuer, 1.0); err != nil {
				return fmt.Errorf("upsert XLM price: %w", err)
			}
			continue
		}

		price, err := s.fetchAssetPrice(ctx, asset.Code, asset.Issuer)
		if err != nil {
			slog.Warn("failed to fetch price for asset", "code", asset.Code, "issuer", asset.Issuer, "error", err)
			continue
		}

		if err := s.repo.UpsertTokenPrice(ctx, asset.Code, asset.Issuer, price); err != nil {
			return fmt.Errorf("upsert token price: %w", err)
		}

		slog.Debug("fetched price", "asset", asset.Code, "price_xlm", price)
	}

	return nil
}

// fetchAssetPrice gets the XLM price for an asset from the SDEX orderbook.
func (s *Syncer) fetchAssetPrice(ctx context.Context, code, issuer string) (float64, error) {
	// Build the orderbook request - we want to sell the asset for XLM
	req := horizonclient.OrderBookRequest{
		SellingAssetType:   horizonclient.AssetType(getAssetType(code)),
		SellingAssetCode:   code,
		SellingAssetIssuer: issuer,
		BuyingAssetType:    horizonclient.AssetTypeNative,
		Limit:              1,
	}

	orderbook, err := s.horizon.OrderBook(req)
	if err != nil {
		return 0, fmt.Errorf("fetch orderbook: %w", err)
	}

	// Take the best bid (highest price someone is willing to pay in XLM)
	if len(orderbook.Bids) == 0 {
		return 0, fmt.Errorf("no bids in orderbook")
	}

	price, err := strconv.ParseFloat(orderbook.Bids[0].Price, 64)
	if err != nil {
		return 0, fmt.Errorf("parse bid price: %w", err)
	}

	return price, nil
}

// getAssetType returns the Stellar asset type string for an asset code.
func getAssetType(code string) string {
	if len(code) <= 4 {
		return "credit_alphanum4"
	}
	return "credit_alphanum12"
}
