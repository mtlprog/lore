package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtlprog/lore/internal/handler/mocks"
	"github.com/mtlprog/lore/internal/model"
	"github.com/stretchr/testify/mock"
)

func TestHandler_Reputation_EmptyAccountID(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountQuerier(t)
	reputation := mocks.NewMockReputationQuerier(t)
	tmpl := mocks.NewMockTemplateRenderer(t)

	h, err := New(stellar, accounts, reputation, tmpl)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/accounts//reputation", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	h.Reputation(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandler_Reputation_ServiceUnavailable(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountQuerier(t)
	tmpl := mocks.NewMockTemplateRenderer(t)

	// Create handler with nil reputation service
	h, err := New(stellar, accounts, nil, tmpl)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123/reputation", nil)
	req.SetPathValue("id", "GABC123")
	w := httptest.NewRecorder()

	h.Reputation(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandler_Reputation_GetGraphError(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountQuerier(t)
	reputation := mocks.NewMockReputationQuerier(t)
	tmpl := mocks.NewMockTemplateRenderer(t)

	reputation.EXPECT().
		GetGraph(mock.Anything, "GABC123").
		Return(nil, errors.New("database error"))

	h, err := New(stellar, accounts, reputation, tmpl)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123/reputation", nil)
	req.SetPathValue("id", "GABC123")
	w := httptest.NewRecorder()

	h.Reputation(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_Reputation_Success(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountQuerier(t)
	reputation := mocks.NewMockReputationQuerier(t)
	tmpl := mocks.NewMockTemplateRenderer(t)

	graph := &model.ReputationGraph{
		Score: &model.ReputationScore{
			Grade:         "A",
			WeightedScore: 4.0,
			TotalRatings:  3,
		},
	}

	reputation.EXPECT().
		GetGraph(mock.Anything, "GABC123").
		Return(graph, nil)

	accounts.EXPECT().
		GetAccountNames(mock.Anything, []string{"GABC123"}).
		Return(map[string]string{"GABC123": "Test Account"}, nil)

	tmpl.EXPECT().
		Render(mock.Anything, "reputation.html", mock.MatchedBy(func(data ReputationData) bool {
			return data.AccountID == "GABC123" &&
				data.AccountName == "Test Account" &&
				data.Score != nil &&
				data.Score.Grade == "A"
		})).
		Return(nil)

	h, err := New(stellar, accounts, reputation, tmpl)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123/reputation", nil)
	req.SetPathValue("id", "GABC123")
	w := httptest.NewRecorder()

	h.Reputation(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHandler_Reputation_NoGraph_FallsBackToScore(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountQuerier(t)
	reputation := mocks.NewMockReputationQuerier(t)
	tmpl := mocks.NewMockTemplateRenderer(t)

	// GetGraph returns nil (no graph data)
	reputation.EXPECT().
		GetGraph(mock.Anything, "GABC123").
		Return(nil, nil)

	// Falls back to GetScore
	score := &model.ReputationScore{
		Grade:         "B",
		WeightedScore: 3.0,
		TotalRatings:  2,
	}
	reputation.EXPECT().
		GetScore(mock.Anything, "GABC123").
		Return(score, nil)

	accounts.EXPECT().
		GetAccountNames(mock.Anything, []string{"GABC123"}).
		Return(map[string]string{}, nil)

	tmpl.EXPECT().
		Render(mock.Anything, "reputation.html", mock.MatchedBy(func(data ReputationData) bool {
			return data.AccountID == "GABC123" &&
				data.AccountName == "GABC123" && // Falls back to ID when name not found
				data.Score != nil &&
				data.Score.Grade == "B" &&
				data.Graph == nil
		})).
		Return(nil)

	h, err := New(stellar, accounts, reputation, tmpl)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123/reputation", nil)
	req.SetPathValue("id", "GABC123")
	w := httptest.NewRecorder()

	h.Reputation(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHandler_Reputation_TemplateRenderError(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountQuerier(t)
	reputation := mocks.NewMockReputationQuerier(t)
	tmpl := mocks.NewMockTemplateRenderer(t)

	reputation.EXPECT().
		GetGraph(mock.Anything, "GABC123").
		Return(&model.ReputationGraph{}, nil)

	accounts.EXPECT().
		GetAccountNames(mock.Anything, []string{"GABC123"}).
		Return(map[string]string{}, nil)

	tmpl.EXPECT().
		Render(mock.Anything, "reputation.html", mock.Anything).
		Return(errors.New("template error"))

	h, err := New(stellar, accounts, reputation, tmpl)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123/reputation", nil)
	req.SetPathValue("id", "GABC123")
	w := httptest.NewRecorder()

	h.Reputation(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// Ensure Render receives a bytes.Buffer
func TestHandler_Reputation_RendersToBuffer(t *testing.T) {
	stellar := mocks.NewMockStellarServicer(t)
	accounts := mocks.NewMockAccountQuerier(t)
	reputation := mocks.NewMockReputationQuerier(t)
	tmpl := mocks.NewMockTemplateRenderer(t)

	reputation.EXPECT().
		GetGraph(mock.Anything, "GABC123").
		Return(&model.ReputationGraph{}, nil)

	accounts.EXPECT().
		GetAccountNames(mock.Anything, []string{"GABC123"}).
		Return(map[string]string{}, nil)

	tmpl.EXPECT().
		Render(mock.MatchedBy(func(w interface{}) bool {
			_, ok := w.(*bytes.Buffer)
			return ok
		}), "reputation.html", mock.Anything).
		Return(nil)

	h, err := New(stellar, accounts, reputation, tmpl)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/accounts/GABC123/reputation", nil)
	req.SetPathValue("id", "GABC123")
	w := httptest.NewRecorder()

	h.Reputation(w, req)
}
