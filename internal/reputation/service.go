package reputation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mtlprog/lore/internal/model"
)

// Service provides reputation data for the handler layer.
type Service struct {
	repo    *Repository
	builder *GraphBuilder
}

// NewService creates a new reputation service.
func NewService(pool *pgxpool.Pool) (*Service, error) {
	repo, err := NewRepository(pool)
	if err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}

	calc := NewCalculator()
	builder, err := NewGraphBuilder(repo, calc)
	if err != nil {
		return nil, fmt.Errorf("create graph builder: %w", err)
	}

	return &Service{
		repo:    repo,
		builder: builder,
	}, nil
}

// GetScore returns the reputation score for an account.
func (s *Service) GetScore(ctx context.Context, accountID string) (*model.ReputationScore, error) {
	score, err := s.repo.GetScore(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get score: %w", err)
	}

	if score == nil || score.TotalRatings == 0 {
		return nil, nil
	}

	return &model.ReputationScore{
		WeightedScore: score.WeightedScore,
		BaseScore:     score.BaseScore,
		Grade:         score.Grade(),
		RatingCountA:  score.RatingCountA,
		RatingCountB:  score.RatingCountB,
		RatingCountC:  score.RatingCountC,
		RatingCountD:  score.RatingCountD,
		TotalRatings:  score.TotalRatings,
		TotalWeight:   score.TotalWeight,
	}, nil
}

// GetGraph returns the reputation graph for an account.
func (s *Service) GetGraph(ctx context.Context, accountID string) (*model.ReputationGraph, error) {
	graph, err := s.builder.BuildGraph(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	if graph == nil || (len(graph.Level1Nodes) == 0 && len(graph.Level2Nodes) == 0) {
		return nil, nil
	}

	// Convert internal types to model types
	result := &model.ReputationGraph{
		TargetAccountID: graph.TargetAccountID,
		TargetName:      graph.TargetName,
		Level1Nodes:     make([]model.ReputationNode, len(graph.Level1Nodes)),
		Level2Nodes:     make([]model.ReputationNode, len(graph.Level2Nodes)),
	}

	// Convert score
	if graph.Score != nil && graph.Score.TotalRatings > 0 {
		result.Score = &model.ReputationScore{
			WeightedScore: graph.Score.WeightedScore,
			BaseScore:     graph.Score.BaseScore,
			Grade:         graph.Score.Grade(),
			RatingCountA:  graph.Score.RatingCountA,
			RatingCountB:  graph.Score.RatingCountB,
			RatingCountC:  graph.Score.RatingCountC,
			RatingCountD:  graph.Score.RatingCountD,
			TotalRatings:  graph.Score.TotalRatings,
			TotalWeight:   graph.Score.TotalWeight,
		}
	}

	// Convert level 1 nodes
	for i, node := range graph.Level1Nodes {
		result.Level1Nodes[i] = model.ReputationNode{
			AccountID:    node.AccountID,
			Name:         node.Name,
			Rating:       node.Rating,
			Weight:       node.Weight,
			PortfolioXLM: node.PortfolioXLM,
			Connections:  node.Connections,
			OwnScore:     node.OwnScore,
			Distance:     node.Distance,
		}
	}

	// Convert level 2 nodes
	for i, node := range graph.Level2Nodes {
		result.Level2Nodes[i] = model.ReputationNode{
			AccountID:    node.AccountID,
			Name:         node.Name,
			Rating:       node.Rating,
			Weight:       node.Weight,
			PortfolioXLM: node.PortfolioXLM,
			Connections:  node.Connections,
			OwnScore:     node.OwnScore,
			Distance:     node.Distance,
		}
	}

	return result, nil
}
