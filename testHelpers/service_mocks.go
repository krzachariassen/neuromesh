package testHelpers

import (
	"context"
	"errors"
	"time"

	conversationApp "neuromesh/internal/conversation/application"
	conversationDomain "neuromesh/internal/conversation/domain"
	planningDomain "neuromesh/internal/planning/domain"
	userApp "neuromesh/internal/user/application"
	userDomain "neuromesh/internal/user/domain"

	"github.com/stretchr/testify/mock"
)

// MockConversationService provides a testify-based mock for conversation service operations
type MockConversationService struct {
	mock.Mock
}

// NewMockConversationService creates a new mock conversation service instance
func NewMockConversationService() *MockConversationService {
	return &MockConversationService{}
}

func (m *MockConversationService) CreateConversation(ctx context.Context, id, sessionID, userID string) (*conversationDomain.Conversation, error) {
	args := m.Called(ctx, id, sessionID, userID)
	return args.Get(0).(*conversationDomain.Conversation), args.Error(1)
}

func (m *MockConversationService) GetConversation(ctx context.Context, conversationID string) (*conversationDomain.Conversation, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).(*conversationDomain.Conversation), args.Error(1)
}

func (m *MockConversationService) GetConversationWithMessages(ctx context.Context, conversationID string) (*conversationDomain.Conversation, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).(*conversationDomain.Conversation), args.Error(1)
}

func (m *MockConversationService) UpdateConversationStatus(ctx context.Context, conversationID string, status conversationDomain.ConversationStatus) error {
	args := m.Called(ctx, conversationID, status)
	return args.Error(0)
}

func (m *MockConversationService) DeleteConversation(ctx context.Context, conversationID string) error {
	args := m.Called(ctx, conversationID)
	return args.Error(0)
}

func (m *MockConversationService) GetConversationMessages(ctx context.Context, conversationID string) ([]conversationDomain.ConversationMessage, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).([]conversationDomain.ConversationMessage), args.Error(1)
}

func (m *MockConversationService) GetMessagesByRole(ctx context.Context, conversationID string, role conversationDomain.MessageRole) ([]conversationDomain.ConversationMessage, error) {
	args := m.Called(ctx, conversationID, role)
	return args.Get(0).([]conversationDomain.ConversationMessage), args.Error(1)
}

func (m *MockConversationService) LinkConversationToSession(ctx context.Context, conversationID, sessionID string) error {
	args := m.Called(ctx, conversationID, sessionID)
	return args.Error(0)
}

func (m *MockConversationService) LinkConversationToUser(ctx context.Context, conversationID, userID string) error {
	args := m.Called(ctx, conversationID, userID)
	return args.Error(0)
}

func (m *MockConversationService) FindConversationsByUser(ctx context.Context, userID string) ([]*conversationDomain.Conversation, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*conversationDomain.Conversation), args.Error(1)
}

func (m *MockConversationService) FindConversationsBySession(ctx context.Context, sessionID string) ([]*conversationDomain.Conversation, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]*conversationDomain.Conversation), args.Error(1)
}

func (m *MockConversationService) FindActiveConversations(ctx context.Context) ([]*conversationDomain.Conversation, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*conversationDomain.Conversation), args.Error(1)
}

func (m *MockConversationService) GetAllConversations(ctx context.Context) ([]*conversationDomain.Conversation, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*conversationDomain.Conversation), args.Error(1)
}

func (m *MockConversationService) AddMessage(ctx context.Context, conversationID, messageID string, role conversationDomain.MessageRole, content string, metadata map[string]interface{}) error {
	args := m.Called(ctx, conversationID, messageID, role, content, metadata)
	return args.Error(0)
}

func (m *MockConversationService) LinkExecutionPlan(ctx context.Context, conversationID, executionPlanID string) error {
	args := m.Called(ctx, conversationID, executionPlanID)
	return args.Error(0)
}

func (m *MockConversationService) EnsureSchema(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockUserService provides a testify-based mock for user service operations
type MockUserService struct {
	mock.Mock
}

// NewMockUserService creates a new mock user service instance
func NewMockUserService() *MockUserService {
	return &MockUserService{}
}

func (m *MockUserService) CreateUser(ctx context.Context, userID, sessionID string, userType userDomain.UserType) (*userDomain.User, error) {
	args := m.Called(ctx, userID, sessionID, userType)
	return args.Get(0).(*userDomain.User), args.Error(1)
}

func (m *MockUserService) GetUser(ctx context.Context, userID string) (*userDomain.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*userDomain.User), args.Error(1)
}

func (m *MockUserService) GetUserWithSessions(ctx context.Context, userID string) (*userDomain.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*userDomain.User), args.Error(1)
}

func (m *MockUserService) UpdateUserStatus(ctx context.Context, userID string, status userDomain.UserStatus) error {
	args := m.Called(ctx, userID, status)
	return args.Error(0)
}

func (m *MockUserService) UpdateUserLastSeen(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) SetUserMetadata(ctx context.Context, userID, key string, value interface{}) error {
	args := m.Called(ctx, userID, key, value)
	return args.Error(0)
}

func (m *MockUserService) DeleteUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) CreateSession(ctx context.Context, sessionID, userID string, duration time.Duration) (*userDomain.Session, error) {
	args := m.Called(ctx, sessionID, userID, duration)
	return args.Get(0).(*userDomain.Session), args.Error(1)
}

func (m *MockUserService) GetSession(ctx context.Context, sessionID string) (*userDomain.Session, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).(*userDomain.Session), args.Error(1)
}

func (m *MockUserService) GetUserSessions(ctx context.Context, userID string) ([]*userDomain.Session, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*userDomain.Session), args.Error(1)
}

func (m *MockUserService) ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error {
	args := m.Called(ctx, sessionID, duration)
	return args.Error(0)
}

func (m *MockUserService) CloseSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockUserService) CleanupExpiredSessions(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUserService) FindUsersByType(ctx context.Context, userType userDomain.UserType) ([]*userDomain.User, error) {
	args := m.Called(ctx, userType)
	return args.Get(0).([]*userDomain.User), args.Error(1)
}

func (m *MockUserService) FindActiveUsers(ctx context.Context) ([]*userDomain.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*userDomain.User), args.Error(1)
}

func (m *MockUserService) EnsureSchema(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Ensure mocks implement the interfaces
var _ conversationApp.ConversationService = (*MockConversationService)(nil)
var _ userApp.UserService = (*MockUserService)(nil)
var _ conversationDomain.ConversationGraphService = (*MockGraphService)(nil)

// MockGraphService provides a testify-based mock for graph service operations
type MockGraphService struct {
	mock.Mock
}

// NewMockGraphService creates a new mock graph service instance
func NewMockGraphService() *MockGraphService {
	return &MockGraphService{}
}

func (m *MockGraphService) GetConversationGraph(ctx context.Context, conversationID string) (*conversationDomain.GraphData, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).(*conversationDomain.GraphData), args.Error(1)
}

func (m *MockGraphService) GetConversationSubgraph(ctx context.Context, conversationID string, nodeTypes []string) (*conversationDomain.GraphData, error) {
	args := m.Called(ctx, conversationID, nodeTypes)
	return args.Get(0).(*conversationDomain.GraphData), args.Error(1)
}

func (m *MockGraphService) GetGraphStats(ctx context.Context, conversationID string) (map[string]interface{}, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockPlanningResultRepository provides a testify-based mock for planning result repository operations
type MockPlanningResultRepository struct {
	// Control behavior
	ShouldFailStore bool

	// Track calls
	StoreCalled               bool
	LinkToRequestCalled       bool
	LinkToExecutionPlanCalled bool

	// Capture arguments
	StoredResult           *planningDomain.PlanningResult
	LinkedPlanningResultID string
	LinkedRequestID        string
	LinkedExecutionPlanID  string

	// Return values
	StoredResults []*planningDomain.PlanningResult
}

// NewMockPlanningResultRepository creates a new mock planning result repository
func NewMockPlanningResultRepository() *MockPlanningResultRepository {
	return &MockPlanningResultRepository{
		StoredResults: make([]*planningDomain.PlanningResult, 0),
	}
}

func (m *MockPlanningResultRepository) Store(ctx context.Context, result *planningDomain.PlanningResult) error {
	m.StoreCalled = true
	m.StoredResult = result
	m.StoredResults = append(m.StoredResults, result)

	if m.ShouldFailStore {
		return errors.New("mock store error")
	}
	return nil
}

func (m *MockPlanningResultRepository) GetByID(ctx context.Context, id string) (*planningDomain.PlanningResult, error) {
	for _, result := range m.StoredResults {
		if result.ID == id {
			return result, nil
		}
	}
	return nil, errors.New("planning result not found")
}

func (m *MockPlanningResultRepository) GetByRequestID(ctx context.Context, requestID string) ([]*planningDomain.PlanningResult, error) {
	var results []*planningDomain.PlanningResult
	for _, result := range m.StoredResults {
		if result.RequestID == requestID {
			results = append(results, result)
		}
	}
	return results, nil
}

func (m *MockPlanningResultRepository) Update(ctx context.Context, result *planningDomain.PlanningResult) error {
	return nil
}

func (m *MockPlanningResultRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *MockPlanningResultRepository) LinkToRequest(ctx context.Context, planningResultID, requestID string) error {
	m.LinkToRequestCalled = true
	m.LinkedPlanningResultID = planningResultID
	m.LinkedRequestID = requestID
	return nil
}

func (m *MockPlanningResultRepository) LinkToExecutionPlan(ctx context.Context, planningResultID, executionPlanID string) error {
	m.LinkToExecutionPlanCalled = true
	m.LinkedPlanningResultID = planningResultID
	m.LinkedExecutionPlanID = executionPlanID
	return nil
}

func (m *MockPlanningResultRepository) LinkToConversation(ctx context.Context, planningResultID, conversationID string) error {
	// Add tracking fields if needed for specific tests
	return nil
}
