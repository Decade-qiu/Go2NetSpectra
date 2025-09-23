package model

import (
	"context"
)

// Analyzer defines the standard interface for an AI analyzer.
type Analyzer interface {
	// AnalyzeTraffic receives a text input and returns the analysis result from the AI model.
	AnalyzeTraffic(ctx context.Context, input string) (string, error)
}
