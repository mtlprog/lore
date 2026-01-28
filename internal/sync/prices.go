package sync

import (
	"context"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
	"github.com/stellar/go/clients/horizonclient"
)

// syncTokenPrices fetches token prices from SDEX and stores them.
// Returns a list of failed asset fetches (code:issuer format) and any critical error.
func (s *Syncer) syncTokenPrices(ctx context.Context) ([]string, error) {
	assets, err := s.repo.GetUniqueAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("get unique assets: %w", err)
	}

	s.logger.Debug("fetching prices for assets", "count", len(assets))

	var failedAssets []string

	for _, asset := range assets {
		// Skip native XLM - its price is always 1 XLM
		if asset.Code == "XLM" && asset.Issuer == "" {
			if err := s.repo.UpsertTokenPrice(ctx, asset.Code, asset.Issuer, decimal.NewFromInt(1)); err != nil {
				return failedAssets, fmt.Errorf("upsert XLM price: %w", err)
			}
			continue
		}

		price, err := s.fetchAssetPrice(ctx, asset.Code, asset.Issuer)
		if err != nil {
			s.logger.Error("failed to fetch price",
				"code", asset.Code,
				"issuer", asset.Issuer,
				"error", err,
			)
			failedAssets = append(failedAssets, fmt.Sprintf("%s:%s", asset.Code, asset.Issuer))
			continue
		}

		if err := s.repo.UpsertTokenPrice(ctx, asset.Code, asset.Issuer, price); err != nil {
			return failedAssets, fmt.Errorf("upsert token price: %w", err)
		}

		s.logger.Debug("fetched price", "asset", asset.Code, "price_xlm", price)
	}

	if len(failedAssets) > 0 {
		s.logger.Error("price fetch failures",
			"failed_assets", failedAssets,
			"count", len(failedAssets),
		)
	}

	return failedAssets, nil
}

// fetchAssetPrice gets the XLM price for an asset from the SDEX orderbook.
func (s *Syncer) fetchAssetPrice(_ context.Context, code, issuer string) (decimal.Decimal, error) {
	req := horizonclient.OrderBookRequest{
		SellingAssetType:   horizonclient.AssetType(getAssetType(code)),
		SellingAssetCode:   code,
		SellingAssetIssuer: issuer,
		BuyingAssetType:    horizonclient.AssetTypeNative,
		Limit:              1,
	}

	orderbook, err := s.horizon.OrderBook(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("fetch orderbook: %w", err)
	}

	if len(orderbook.Bids) == 0 {
		return decimal.Zero, fmt.Errorf("no bids in orderbook")
	}

	price, err := strconv.ParseFloat(orderbook.Bids[0].Price, 64)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse bid price: %w", err)
	}

	return decimal.NewFromFloat(price), nil
}

// getAssetType returns the Stellar asset type string for an asset code.
func getAssetType(code string) string {
	if len(code) <= 4 {
		return "credit_alphanum4"
	}
	return "credit_alphanum12"
}
