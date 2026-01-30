package reputation

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
)

// GraphBuilder constructs 2-level reputation graphs.
type GraphBuilder struct {
	repo *Repository
	calc *Calculator
}

// NewGraphBuilder creates a new graph builder.
func NewGraphBuilder(repo *Repository, calc *Calculator) (*GraphBuilder, error) {
	if repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if calc == nil {
		calc = NewCalculator()
	}
	return &GraphBuilder{
		repo: repo,
		calc: calc,
	}, nil
}

// BuildGraph constructs a 2-level reputation graph for the target account.
// Level 1: Direct raters (who gave A/B/C/D to this account)
// Level 2: Raters of the Level 1 raters
func (g *GraphBuilder) BuildGraph(ctx context.Context, targetAccountID string) (*Graph, error) {
	// Get target account name
	targetName, err := g.repo.GetAccountName(ctx, targetAccountID)
	if err != nil {
		return nil, fmt.Errorf("get target name: %w", err)
	}

	// Get reputation score
	score, err := g.repo.GetScore(ctx, targetAccountID)
	if err != nil {
		return nil, fmt.Errorf("get score: %w", err)
	}

	// Get direct raters (level 1)
	directRaters, err := g.repo.GetDirectRaters(ctx, targetAccountID)
	if err != nil {
		return nil, fmt.Errorf("get direct raters: %w", err)
	}

	// Convert to GraphNodes
	level1Nodes := make([]GraphNode, 0, len(directRaters))
	level1IDs := make([]string, 0, len(directRaters))

	// Get connection counts for weight calculation
	connections, err := g.repo.GetConnectionCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection counts: %w", err)
	}

	for _, rater := range directRaters {
		weight := g.calc.CalculateRaterWeight(rater.PortfolioXLM, connections[rater.AccountID])
		portfolioFloat, exact := rater.PortfolioXLM.Float64()
		if !exact {
			slog.Debug("precision loss converting portfolio to float64", "account_id", rater.AccountID, "portfolio", rater.PortfolioXLM.String())
		}

		level1Nodes = append(level1Nodes, GraphNode{
			AccountID:    rater.AccountID,
			Name:         rater.Name,
			Rating:       rater.Rating,
			Weight:       weight,
			PortfolioXLM: portfolioFloat,
			Connections:  connections[rater.AccountID],
			OwnScore:     rater.OwnScore,
			Distance:     1,
		})
		level1IDs = append(level1IDs, rater.AccountID)
	}

	// Get raters of raters (level 2)
	ratersOfRaters, err := g.repo.GetRatersOfRaters(ctx, level1IDs, []string{targetAccountID})
	if err != nil {
		return nil, fmt.Errorf("get raters of raters: %w", err)
	}

	// Convert to GraphNodes and deduplicate
	seen := make(map[string]bool)
	for _, id := range level1IDs {
		seen[id] = true
	}
	seen[targetAccountID] = true

	level2Nodes := make([]GraphNode, 0, len(ratersOfRaters))
	for _, rater := range ratersOfRaters {
		if seen[rater.AccountID] {
			continue
		}
		seen[rater.AccountID] = true

		weight := g.calc.CalculateRaterWeight(rater.PortfolioXLM, connections[rater.AccountID])
		portfolioFloat, exact := rater.PortfolioXLM.Float64()
		if !exact {
			slog.Debug("precision loss converting portfolio to float64", "account_id", rater.AccountID, "portfolio", rater.PortfolioXLM.String())
		}

		level2Nodes = append(level2Nodes, GraphNode{
			AccountID:    rater.AccountID,
			Name:         rater.Name,
			Rating:       rater.Rating,
			Weight:       weight,
			PortfolioXLM: portfolioFloat,
			Connections:  connections[rater.AccountID],
			OwnScore:     rater.OwnScore,
			Distance:     2,
		})
	}

	// Sort nodes by rating (A first), then by weight (highest first)
	sortNodes := func(nodes []GraphNode) {
		sort.Slice(nodes, func(i, j int) bool {
			// Sort by rating first (A > B > C > D)
			if ratingPriority(nodes[i].Rating) != ratingPriority(nodes[j].Rating) {
				return ratingPriority(nodes[i].Rating) > ratingPriority(nodes[j].Rating)
			}
			// Then by weight (highest first)
			return nodes[i].Weight > nodes[j].Weight
		})
	}
	sortNodes(level1Nodes)
	sortNodes(level2Nodes)

	return &Graph{
		TargetAccountID: targetAccountID,
		TargetName:      targetName,
		Score:           score,
		Level1Nodes:     level1Nodes,
		Level2Nodes:     level2Nodes,
	}, nil
}

// ratingPriority returns the sort priority for a rating (higher = better).
func ratingPriority(rating Rating) int {
	return int(rating.Value())
}
