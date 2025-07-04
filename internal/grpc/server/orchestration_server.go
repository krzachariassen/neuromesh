package server

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"neuromesh/internal/agent/domain"
	pb "neuromesh/internal/api/grpc/api"
	"neuromesh/internal/logging"
	"neuromesh/internal/messaging"
)

// OrchestrationServer implements the gRPC OrchestrationService as a stateless proxy.
// It delegates:
// - Agent registration/unregistration to the registry service (domain logic)
// - Message streaming to the AI Message Bus (communication)
// It contains NO AI logic or business logic.
type OrchestrationServer struct {
	pb.UnimplementedOrchestrationServiceServer

	messageBus      messaging.AIMessageBus
	registryService domain.AgentRegistry
	logger          logging.Logger

	// Track active streams for cleanup
	activeStreams map[string]context.CancelFunc
	streamsMutex  sync.RWMutex
}

// NewOrchestrationServer creates a new gRPC server that acts as a stateless proxy
func NewOrchestrationServer(messageBus messaging.AIMessageBus, registryService domain.AgentRegistry, logger logging.Logger) *OrchestrationServer {
	return &OrchestrationServer{
		messageBus:      messageBus,
		registryService: registryService,
		logger:          logger,
		activeStreams:   make(map[string]context.CancelFunc),
	}
}

// RegisterAgent delegates agent registration to the registry service (domain logic)
func (s *OrchestrationServer) RegisterAgent(ctx context.Context, req *pb.RegisterAgentRequest) (*pb.RegisterAgentResponse, error) {
	// Input validation
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	if req.AgentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "agent ID cannot be empty")
	}

	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "agent name cannot be empty")
	}

	if len(req.Capabilities) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "agent must have at least one capability")
	}

	s.logger.Info("Registering agent via gRPC",
		"agent_id", req.AgentId,
		"capabilities", req.Capabilities)

	// Convert gRPC message to internal domain.Agent format
	agent := &domain.Agent{
		ID:           req.AgentId,
		Name:         req.Name,
		Description:  "Agent registered via gRPC",
		Capabilities: convertCapabilitiesFromPb(req.Capabilities),
		Status:       domain.AgentStatusOnline,
		Metadata:     convertStructToStringMap(req.Metadata),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastSeen:     time.Now(),
	}

	// Delegate to registry service (domain logic)
	err := s.registryService.RegisterAgent(ctx, agent)
	if err != nil {
		s.logger.Error("Failed to register agent", err,
			"agent_id", req.AgentId)
		return nil, status.Errorf(codes.Internal, "failed to register agent: %v", err)
	}

	// Prepare agent's message queue and routing (without starting consumption)
	// This ensures the agent can receive messages when it opens a conversation
	err = s.messageBus.PrepareAgentQueue(ctx, req.AgentId)
	if err != nil {
		s.logger.Error("Failed to prepare agent queue", err,
			"agent_id", req.AgentId)
		// Note: We don't fail the registration since the agent is already in the graph
		// The agent can still be used, but won't receive messages until this is fixed
		s.logger.Warn("Agent registered but queue not prepared",
			"agent_id", req.AgentId)
	} else {
		s.logger.Info("Agent queue prepared successfully",
			"agent_id", req.AgentId)
	}

	s.logger.Info("Successfully registered agent",
		"agent_id", req.AgentId)

	return &pb.RegisterAgentResponse{
		Success:      true,
		Message:      "Agent registered successfully",
		RegisteredAt: timestamppb.Now(),
	}, nil
}

// UnregisterAgent delegates agent unregistration to the registry service (domain logic)
func (s *OrchestrationServer) UnregisterAgent(ctx context.Context, req *pb.UnregisterAgentRequest) (*pb.UnregisterAgentResponse, error) {
	// Input validation
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	if req.AgentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "agent ID cannot be empty")
	}

	s.logger.Info("Unregistering agent via gRPC",
		"agent_id", req.AgentId,
		"reason", req.Reason)

	// Clean up any active streams for this agent
	s.streamsMutex.Lock()
	if cancel, exists := s.activeStreams[req.AgentId]; exists {
		cancel()
		delete(s.activeStreams, req.AgentId)
	}
	s.streamsMutex.Unlock()

	// Delegate to registry service (domain logic)
	err := s.registryService.UnregisterAgent(ctx, req.AgentId)
	if err != nil {
		s.logger.Error("Failed to unregister agent", err,
			"agent_id", req.AgentId)
		return nil, status.Errorf(codes.Internal, "failed to unregister agent: %v", err)
	}

	// TODO: Add message bus cleanup when AIMessageBus supports Unsubscribe
	s.logger.Info("Successfully unregistered agent",
		"agent_id", req.AgentId)

	return &pb.UnregisterAgentResponse{
		Success: true,
		Message: "Agent unregistered successfully",
	}, nil
}

// UpdateAgentStatus handles agent status updates - pure infrastructure endpoint
func (s *OrchestrationServer) UpdateAgentStatus(ctx context.Context, req *pb.UpdateAgentStatusRequest) (*pb.UpdateAgentStatusResponse, error) {
	// Input validation
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	if req.AgentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "agent ID cannot be empty")
	}

	s.logger.Debug("Updating agent status via dedicated endpoint",
		"agent_id", req.AgentId,
		"status", req.Status)

	// Convert protobuf status to domain status
	var domainStatus domain.AgentStatus
	switch req.Status {
	case pb.AgentStatus_AGENT_STATUS_HEALTHY:
		domainStatus = domain.AgentStatusOnline
	case pb.AgentStatus_AGENT_STATUS_BUSY:
		domainStatus = domain.AgentStatusBusy
	case pb.AgentStatus_AGENT_STATUS_ERROR:
		domainStatus = domain.AgentStatusError
	case pb.AgentStatus_AGENT_STATUS_SHUTTING_DOWN:
		domainStatus = domain.AgentStatusShuttingDown
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid agent status: %v", req.Status)
	}

	// Update status in registry
	err := s.registryService.UpdateAgentStatus(ctx, req.AgentId, domainStatus)
	if err != nil {
		s.logger.Error("Failed to update agent status", err,
			"agent_id", req.AgentId,
			"status", req.Status)
		return nil, status.Errorf(codes.Internal, "failed to update agent status: %v", err)
	}

	// Update last seen timestamp
	err = s.registryService.UpdateAgentLastSeen(ctx, req.AgentId)
	if err != nil {
		s.logger.Warn("Failed to update agent last seen", err,
			"agent_id", req.AgentId)
		// Don't fail the request for this
	}

	s.logger.Debug("Successfully updated agent status",
		"agent_id", req.AgentId,
		"status", req.Status)

	return &pb.UpdateAgentStatusResponse{
		Success:    true,
		Message:    "Agent status updated successfully",
		ServerTime: timestamppb.Now(),
	}, nil
}

// OpenConversation creates a bidirectional stream between the agent and AI Message Bus
func (s *OrchestrationServer) OpenConversation(stream pb.OrchestrationService_OpenConversationServer) error {
	ctx := stream.Context()

	s.logger.Info("Opening conversation stream")

	// Get agent ID from gRPC metadata (no need to wait for identification message!)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "missing gRPC metadata")
	}

	agentIDs := md.Get("agent-id")
	if len(agentIDs) == 0 {
		return status.Errorf(codes.InvalidArgument, "missing agent-id in gRPC metadata")
	}

	agentID := agentIDs[0]
	if agentID == "" {
		return status.Errorf(codes.InvalidArgument, "agent-id cannot be empty")
	}

	s.logger.Info("Agent opened conversation", "agent_id", agentID)

	// Subscribe to message bus for agent communication
	s.logger.Debug("Subscribing to message bus", "agent_id", agentID)
	messageChan, err := s.messageBus.Subscribe(ctx, agentID)
	if err != nil {
		s.logger.Error("Failed to subscribe to message bus", err, "agent_id", agentID)
		return status.Errorf(codes.Internal, "failed to subscribe to message bus: %v", err)
	}

	// Track this stream for cleanup
	streamCtx, cancel := context.WithCancel(ctx)
	s.streamsMutex.Lock()
	s.activeStreams[agentID] = cancel
	s.streamsMutex.Unlock()

	// Cleanup on exit
	defer func() {
		s.streamsMutex.Lock()
		if _, exists := s.activeStreams[agentID]; exists {
			cancel()
			delete(s.activeStreams, agentID)
		}
		s.streamsMutex.Unlock()
		s.logger.Info("Conversation stream closed", "agent_id", agentID)
	}()

	// Channel for incoming messages from the stream
	incomingChan := make(chan *pb.ConversationMessage, 10)
	errorChan := make(chan error, 1)

	// Goroutine to receive messages from the stream
	go func() {
		defer close(incomingChan)
		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					s.logger.Debug("Stream closed by client", "agent_id", agentID)
					return
				}
				errorChan <- err
				return
			}

			select {
			case incomingChan <- msg:
			case <-streamCtx.Done():
				return
			}
		}
	}()

	// Main event loop - listen for both incoming messages and message bus
	for {
		select {
		case <-streamCtx.Done():
			s.logger.Debug("Stream context cancelled", "agent_id", agentID)
			return nil

		case err := <-errorChan:
			s.logger.Error("Stream error", err, "agent_id", agentID)
			return status.Errorf(codes.Internal, "stream error: %v", err)

		case msg := <-incomingChan:
			if msg == nil {
				// Channel closed, client disconnected
				return nil
			}

			if err := s.processIncomingMessage(streamCtx, msg); err != nil {
				s.logger.Error("Failed to process incoming message", err, "agent_id", agentID)
				// Continue processing other messages
			}

		case busMsg := <-messageChan:
			if busMsg == nil {
				// Message bus closed - this is an error
				return status.Errorf(codes.Internal, "message bus closed")
			}

			// Convert message bus message to protobuf and send to agent
			pbMsg := s.convertToPbMessage(busMsg)
			if err := stream.Send(pbMsg); err != nil {
				s.logger.Error("Failed to send message to agent", err, "agent_id", agentID)
				return status.Errorf(codes.Internal, "failed to send message: %v", err)
			}
		}
	}
}

// processIncomingMessage handles messages received from the agent
func (s *OrchestrationServer) processIncomingMessage(ctx context.Context, msg *pb.ConversationMessage) error {
	s.logger.Debug("Processing incoming message",
		"from_id", msg.FromId,
		"to_id", msg.ToId,
		"type", msg.Type,
		"correlation_id", msg.CorrelationId)

	switch msg.Type {
	case pb.MessageType_MESSAGE_TYPE_INSTRUCTION:
		// AI instruction to agent (shouldn't come from agent, but handle gracefully)
		s.logger.Warn("Received instruction message from agent, this is unexpected",
			"agent_id", msg.FromId)
		return nil

	case pb.MessageType_MESSAGE_TYPE_COMPLETION:
		// Agent reporting completion to AI
		aiMsg := &messaging.AgentToAIMessage{
			AgentID:       msg.FromId,
			Content:       msg.Content,
			MessageType:   messaging.MessageTypeAgentToAI, // Fixed: Use MessageTypeAgentToAI for routing
			CorrelationID: msg.CorrelationId,
			Context:       convertStructToMap(msg.Context),
		}

		return s.messageBus.SendToAI(ctx, aiMsg)

	case pb.MessageType_MESSAGE_TYPE_STATUS_UPDATE:
		// Agent status update
		aiMsg := &messaging.AgentToAIMessage{
			AgentID:       msg.FromId,
			Content:       msg.Content,
			MessageType:   messaging.MessageTypeNotification,
			CorrelationID: msg.CorrelationId,
			Context:       convertStructToMap(msg.Context),
		}

		return s.messageBus.SendToAI(ctx, aiMsg)

	case pb.MessageType_MESSAGE_TYPE_ERROR:
		// Agent error notification
		aiMsg := &messaging.AgentToAIMessage{
			AgentID:       msg.FromId,
			Content:       msg.Content,
			MessageType:   messaging.MessageTypeError,
			CorrelationID: msg.CorrelationId,
			Context:       convertStructToMap(msg.Context),
		}

		return s.messageBus.SendToAI(ctx, aiMsg)

	case pb.MessageType_MESSAGE_TYPE_HEARTBEAT:
		// Agent heartbeat - could be handled separately or ignored in stream
		s.logger.Debug("Received heartbeat in conversation stream", "agent_id", msg.FromId)
		return nil

	default:
		s.logger.Warn("Unknown message type", "type", msg.Type)
		return nil // Don't fail on unknown message types
	}
}

// convertToPbMessage converts internal message to protobuf message
func (s *OrchestrationServer) convertToPbMessage(msg *messaging.Message) *pb.ConversationMessage {
	return &pb.ConversationMessage{
		MessageId:     msg.ID,
		CorrelationId: msg.CorrelationID,
		FromId:        msg.FromID,
		ToId:          msg.ToID,
		Type:          convertMessageType(msg.MessageType),
		Content:       msg.Content,
		Context:       nil, // Simplified for now
		Timestamp:     timestamppb.New(msg.Timestamp),
	}
}

// convertMessageType converts internal message type to protobuf type
func convertMessageType(msgType messaging.MessageType) pb.MessageType {
	switch msgType {
	case messaging.MessageTypeInstruction:
		return pb.MessageType_MESSAGE_TYPE_INSTRUCTION
	case messaging.MessageTypeCompletion:
		return pb.MessageType_MESSAGE_TYPE_COMPLETION
	case messaging.MessageTypeNotification:
		return pb.MessageType_MESSAGE_TYPE_STATUS_UPDATE
	case messaging.MessageTypeError:
		return pb.MessageType_MESSAGE_TYPE_ERROR
	case messaging.MessageTypeClarification:
		return pb.MessageType_MESSAGE_TYPE_STATUS_UPDATE // Map to status update for AI-native approach
	case messaging.MessageTypeAIToAgent:
		return pb.MessageType_MESSAGE_TYPE_INSTRUCTION
	case messaging.MessageTypeAgentToAI:
		return pb.MessageType_MESSAGE_TYPE_COMPLETION
	default:
		return pb.MessageType_MESSAGE_TYPE_UNKNOWN
	}
}

// Helper functions for struct conversion
func convertStructToMap(s interface{}) map[string]interface{} {
	if s == nil {
		return make(map[string]interface{})
	}

	// Check if it's a protobuf Struct
	if pbStruct, ok := s.(*structpb.Struct); ok {
		return pbStruct.AsMap()
	}

	// For other types, return empty map to avoid type errors
	return make(map[string]interface{})
}

func convertStructToStringMap(s interface{}) map[string]string {
	if s == nil {
		return make(map[string]string)
	}

	// Check if it's a protobuf Struct
	if pbStruct, ok := s.(*structpb.Struct); ok {
		result := make(map[string]string)
		for key, value := range pbStruct.AsMap() {
			result[key] = convertValueToString(value)
		}
		return result
	}

	// For other types, return empty map to avoid type errors
	return make(map[string]string)
}

// Helper function to convert any value to string
func convertValueToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	default:
		// For complex types (arrays, objects), convert to string representation
		return fmt.Sprintf("%v", v)
	}
}

// SendInstruction handles AI sending instructions to agents
func (s *OrchestrationServer) SendInstruction(ctx context.Context, req *pb.InstructionMessage) (*pb.InstructionResponse, error) {
	// Input validation
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	if req.AgentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "agent_id cannot be empty")
	}

	if req.Content == "" {
		return nil, status.Errorf(codes.InvalidArgument, "content cannot be empty")
	}

	s.logger.Info("Processing AI instruction to agent",
		"agent_id", req.AgentId,
		"instruction_id", req.InstructionId,
		"capability", req.Capability,
		"correlation_id", req.CorrelationId)

	// Convert instruction to AI message
	aiMsg := &messaging.AgentToAIMessage{
		AgentID:       req.AgentId,
		Content:       req.Content,
		MessageType:   messaging.MessageTypeInstruction,
		CorrelationID: req.CorrelationId,
		Context:       convertStructToMap(req.Parameters),
	}

	err := s.messageBus.SendToAI(ctx, aiMsg)
	if err != nil {
		s.logger.Error("Failed to send AI instruction", err,
			"agent_id", req.AgentId,
			"instruction_id", req.InstructionId)
		return nil, status.Errorf(codes.Internal, "failed to send instruction: %v", err)
	}

	s.logger.Debug("AI instruction sent successfully",
		"agent_id", req.AgentId,
		"instruction_id", req.InstructionId)

	return &pb.InstructionResponse{
		Success:       true,
		Message:       "Instruction sent successfully",
		InstructionId: req.InstructionId,
		CorrelationId: req.CorrelationId,
	}, nil
}

// ReportCompletion handles agents reporting completion of tasks
func (s *OrchestrationServer) ReportCompletion(ctx context.Context, req *pb.CompletionMessage) (*pb.CompletionResponse, error) {
	// Input validation
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	if req.AgentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "agent_id cannot be empty")
	}

	if req.Content == "" {
		return nil, status.Errorf(codes.InvalidArgument, "content cannot be empty")
	}

	s.logger.Info("Processing agent completion report",
		"agent_id", req.AgentId,
		"completion_id", req.CompletionId,
		"instruction_id", req.InstructionId,
		"success", req.Success,
		"correlation_id", req.CorrelationId)

	// Convert completion to AI message
	aiMsg := &messaging.AgentToAIMessage{
		AgentID:       req.AgentId,
		Content:       req.Content,
		MessageType:   messaging.MessageTypeCompletion,
		CorrelationID: req.CorrelationId,
		Context:       convertStructToMap(req.ResultData),
	}

	// If there was an error, include it in the context
	if !req.Success && req.ErrorMessage != "" {
		if aiMsg.Context == nil {
			aiMsg.Context = make(map[string]interface{})
		}
		aiMsg.Context["error"] = req.ErrorMessage
		aiMsg.Context["success"] = false
	}

	err := s.messageBus.SendToAI(ctx, aiMsg)
	if err != nil {
		s.logger.Error("Failed to send completion report", err,
			"agent_id", req.AgentId,
			"completion_id", req.CompletionId)
		return nil, status.Errorf(codes.Internal, "failed to send completion: %v", err)
	}

	s.logger.Debug("Completion report sent successfully",
		"agent_id", req.AgentId,
		"completion_id", req.CompletionId)

	return &pb.CompletionResponse{
		Success:      true,
		Message:      "Completion reported successfully",
		CompletionId: req.CompletionId,
	}, nil
}

func (s *OrchestrationServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	// Input validation
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	if req.AgentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "agent ID is required")
	}

	// Convert protobuf status to string
	statusStr := "healthy"
	switch req.Status {
	case pb.AgentStatus_AGENT_STATUS_HEALTHY:
		statusStr = "healthy"
	case pb.AgentStatus_AGENT_STATUS_BUSY:
		statusStr = "busy"
	case pb.AgentStatus_AGENT_STATUS_ERROR:
		statusStr = "error"
	case pb.AgentStatus_AGENT_STATUS_SHUTTING_DOWN:
		statusStr = "shutting_down"
	default:
		statusStr = "unknown"
	}

	// Update heartbeat in registry - update last seen time
	if err := s.registryService.UpdateAgentLastSeen(ctx, req.AgentId); err != nil {
		if s.logger != nil {
			s.logger.Error("Failed to update agent heartbeat", err, "agent_id", req.AgentId)
		}
		return &pb.HeartbeatResponse{
			Success:    false,
			ServerTime: timestamppb.Now(),
		}, status.Errorf(codes.Internal, "failed to update heartbeat: %v", err)
	}

	if s.logger != nil {
		s.logger.Debug("Agent heartbeat received", "agent_id", req.AgentId, "status", statusStr)
	}

	return &pb.HeartbeatResponse{
		Success:    true,
		ServerTime: timestamppb.Now(),
	}, nil
}

// Helper functions

// convertCapabilitiesFromPb converts protobuf capabilities to domain capabilities
func convertCapabilitiesFromPb(pbCapabilities []*pb.AgentCapability) []domain.AgentCapability {
	capabilities := make([]domain.AgentCapability, len(pbCapabilities))
	for i, cap := range pbCapabilities {
		capabilities[i] = domain.AgentCapability{
			Name:        cap.Name,
			Description: cap.Description,
		}
	}
	return capabilities
}
