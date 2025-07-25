package testHelpers

import (
	"context"

	"neuromesh/internal/execution/domain"

	"github.com/stretchr/testify/mock"
)

// MockResultSynthesizer provides a testify-based mock for result synthesis operations
type MockResultSynthesizer struct {
	mock.Mock
}

// NewMockResultSynthesizer creates a new mock result synthesizer instance
func NewMockResultSynthesizer() *MockResultSynthesizer {
	return &MockResultSynthesizer{}
}

func (m *MockResultSynthesizer) SynthesizeResults(ctx context.Context, planID string) (string, error) {
	args := m.Called(ctx, planID)
	return args.String(0), args.Error(1)
}

func (m *MockResultSynthesizer) GetSynthesisContext(ctx context.Context, planID string) (*domain.SynthesisContext, error) {
	args := m.Called(ctx, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SynthesisContext), args.Error(1)
}
