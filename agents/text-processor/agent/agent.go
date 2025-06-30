// Package agent provides an AI-native text processing agent implementation
package agent

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"unicode"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/ztdp/agents/text-processor/proto/orchestration"
)

// Config holds agent configuration
type Config struct {
	AgentID             string
	Name                string
	OrchestratorAddress string
	ReconnectInterval   time.Duration
}

// AINativeAgent implements the AI-native text processing agent
type AINativeAgent struct {
	config     Config
	client     pb.OrchestrationServiceClient
	conn       *grpc.ClientConn
	sessionID  string
	registered bool
}

// NewAINativeAgent creates a new AI-native agent
func NewAINativeAgent(config Config) *AINativeAgent {
	return &AINativeAgent{
		config: config,
	}
}

// Start connects to the orchestrator and begins operation
func (a *AINativeAgent) Start(ctx context.Context) error {
	log.Printf("🔌 Connecting to orchestrator at %s", a.config.OrchestratorAddress)

	// Connect to orchestrator
	conn, err := grpc.Dial(a.config.OrchestratorAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to orchestrator: %w", err)
	}

	a.conn = conn
	a.client = pb.NewOrchestrationServiceClient(conn)

	// Register with orchestrator
	if err := a.register(ctx); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	// Start conversation stream for receiving instructions
	if err := a.startConversationStream(ctx); err != nil {
		return fmt.Errorf("failed to start conversation stream: %w", err)
	}

	log.Printf("✅ AI-native text processing agent started successfully")
	return nil
}

// Stop gracefully shuts down the agent
func (a *AINativeAgent) Stop(ctx context.Context) error {
	if a.registered {
		_ = a.unregister(ctx)
	}

	if a.conn != nil {
		return a.conn.Close()
	}

	return nil
}

// register registers the agent with the orchestrator
func (a *AINativeAgent) register(ctx context.Context) error {
	capabilities := a.getCapabilities()

	req := &pb.RegisterAgentRequest{
		AgentId:      a.config.AgentID,
		Name:         a.config.Name,
		Type:         "text-processor",
		Capabilities: capabilities,
		Version:      "1.0.0",
	}

	resp, err := a.client.RegisterAgent(ctx, req)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration rejected: %s", resp.Message)
	}

	a.sessionID = resp.SessionId
	a.registered = true

	log.Printf("🎯 Registered with session ID: %s", a.sessionID)
	return nil
}

// unregister unregisters the agent from the orchestrator
func (a *AINativeAgent) unregister(ctx context.Context) error {
	req := &pb.UnregisterAgentRequest{
		AgentId:   a.config.AgentID,
		SessionId: a.sessionID,
		Reason:    "Graceful shutdown",
	}

	_, err := a.client.UnregisterAgent(ctx, req)
	return err
}

// getCapabilities returns the agent's capabilities in the new format
func (a *AINativeAgent) getCapabilities() []*pb.AgentCapability {
	return []*pb.AgentCapability{
		{
			Name:        "word-count",
			Description: "Count the number of words in text",
			Inputs:      []string{"text"},
			Outputs:     []string{"word_count"},
		},
		{
			Name:        "text-analysis",
			Description: "Analyze text properties and characteristics",
			Inputs:      []string{"text"},
			Outputs:     []string{"analysis_report"},
		},
		{
			Name:        "character-count",
			Description: "Count the number of characters in text",
			Inputs:      []string{"text"},
			Outputs:     []string{"character_count"},
		},
	}
}

// ProcessInstruction handles natural language instructions from AI orchestrator
func (a *AINativeAgent) ProcessInstruction(instruction string) string {
	log.Printf("📥 Processing AI instruction: %s", instruction)

	// Extract text from natural language instruction
	text := a.extractTextFromInstruction(instruction)
	log.Printf("📝 Extracted text: '%s'", text)

	// Determine what the AI wants us to do
	instructionLower := strings.ToLower(instruction)

	if strings.Contains(instructionLower, "count") && strings.Contains(instructionLower, "word") {
		count := a.countWords(text)
		response := fmt.Sprintf(`The text "%s" contains %d words.`, text, count)
		log.Printf("✅ Response: %s", response)
		return response
	}

	if strings.Contains(instructionLower, "analyze") || strings.Contains(instructionLower, "analysis") {
		analysis := a.analyzeText(text)
		response := fmt.Sprintf("Analysis of \"%s\": %s", text, analysis)
		log.Printf("✅ Response: %s", response)
		return response
	}

	if strings.Contains(instructionLower, "character") && strings.Contains(instructionLower, "count") {
		count := len(text)
		response := fmt.Sprintf(`The text "%s" contains %d characters.`, text, count)
		log.Printf("✅ Response: %s", response)
		return response
	}

	// Default: word count (most common request)
	count := a.countWords(text)
	response := fmt.Sprintf(`The text "%s" contains %d words.`, text, count)
	log.Printf("✅ Response: %s", response)
	return response
}

// extractTextFromInstruction parses natural language to find text to process
func (a *AINativeAgent) extractTextFromInstruction(instruction string) string {
	// Look for text in quotes
	re := regexp.MustCompile(`["']([^"']+)["']`)
	matches := re.FindStringSubmatch(instruction)
	if len(matches) > 1 {
		return matches[1]
	}

	// Look for "text:" pattern
	if strings.Contains(strings.ToLower(instruction), "text:") {
		parts := strings.Split(instruction, ":")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}

	// Look for "following" pattern
	re = regexp.MustCompile(`following[^:]*:?\s*(.+)`)
	matches = re.FindStringSubmatch(instruction)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Last resort: take everything after common instruction words
	words := strings.Fields(instruction)
	for i, word := range words {
		if strings.ToLower(word) == "in" && i+1 < len(words) {
			return strings.Join(words[i+1:], " ")
		}
	}

	return instruction // Fallback
}

// countWords counts words in text
func (a *AINativeAgent) countWords(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	// Split by whitespace and count non-empty parts
	words := strings.Fields(text)
	return len(words)
}

// analyzeText provides basic text analysis
func (a *AINativeAgent) analyzeText(text string) string {
	if text == "" {
		return "empty text"
	}

	wordCount := a.countWords(text)
	charCount := len(text)

	// Count letters
	letterCount := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			letterCount++
		}
	}

	return fmt.Sprintf("%d words, %d characters, %d letters", wordCount, charCount, letterCount)
}

// createCompletionMessage creates a completion message for the orchestrator
func (a *AINativeAgent) createCompletionMessage(instructionID, correlationID, content string, success bool, errorMsg string) *pb.CompletionMessage {
	completion := &pb.CompletionMessage{
		CompletionId:  fmt.Sprintf("completion-%s-%d", a.config.AgentID, time.Now().Unix()),
		CorrelationId: correlationID,
		InstructionId: instructionID,
		AgentId:       a.config.AgentID,
		Success:       success,
		Content:       content,
		Timestamp:     timestamppb.Now(),
	}

	if !success {
		completion.ErrorMessage = errorMsg
	}

	return completion
}

// StartHeartbeat starts sending heartbeats to the orchestrator every 30 seconds
func (a *AINativeAgent) StartHeartbeat(ctx context.Context, notificationChan chan<- bool) error {
	// Start heartbeat goroutine regardless of connection status
	// In production, this should be called after connection is established
	go a.heartbeatLoop(ctx, notificationChan)

	return nil
}

// heartbeatLoop runs the actual heartbeat loop in a goroutine
func (a *AINativeAgent) heartbeatLoop(ctx context.Context, notificationChan chan<- bool) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Printf("💓 Starting heartbeat loop for agent %s", a.config.AgentID)

	// Send immediate first heartbeat
	a.sendHeartbeat(ctx, notificationChan)

	for {
		select {
		case <-ticker.C:
			a.sendHeartbeat(ctx, notificationChan)
		case <-ctx.Done():
			log.Printf("💓 Heartbeat loop stopped for agent %s", a.config.AgentID)
			return
		}
	}
}

// sendHeartbeat sends a single heartbeat to the orchestrator
func (a *AINativeAgent) sendHeartbeat(ctx context.Context, notificationChan chan<- bool) {
	// Skip actual gRPC call if no connection (for testing)
	if a.client != nil {
		heartbeatReq := &pb.HeartbeatRequest{
			AgentId:   a.config.AgentID,
			SessionId: a.sessionID,
			Status:    pb.AgentStatus_AGENT_STATUS_HEALTHY,
		}

		// Send heartbeat to orchestrator
		_, err := a.client.Heartbeat(ctx, heartbeatReq)
		if err != nil {
			log.Printf("❌ Heartbeat failed for agent %s: %v", a.config.AgentID, err)
			return
		}

		log.Printf("💓 Heartbeat sent for agent %s", a.config.AgentID)
	} else {
		// In test mode or when connection not established
		log.Printf("💓 Heartbeat tick for agent %s (no connection)", a.config.AgentID)
	}

	// Notify test channel if provided
	if notificationChan != nil {
		select {
		case notificationChan <- true:
			// Notification sent successfully
		default:
			// Channel full or closed, continue without blocking
		}
	}
}

// processConversationMessage handles incoming conversation messages and returns appropriate responses
func (a *AINativeAgent) processConversationMessage(msg *pb.ConversationMessage) *pb.ConversationMessage {
	log.Printf("📨 Processing conversation message: %s (type: %v)", msg.MessageId, msg.Type)

	switch msg.Type {
	case pb.MessageType_MESSAGE_TYPE_INSTRUCTION:
		// Process the instruction and create a completion response
		result := a.ProcessInstruction(msg.Content)

		// Create completion message using existing method
		completion := a.createCompletionMessage(msg.MessageId, msg.CorrelationId, result, true, "")

		// Convert to conversation message format
		return &pb.ConversationMessage{
			MessageId:     completion.CompletionId,
			CorrelationId: msg.CorrelationId,
			FromId:        a.config.AgentID,
			ToId:          "orchestrator",
			Type:          pb.MessageType_MESSAGE_TYPE_COMPLETION,
			Content:       completion.Content,
			Context:       completion.ResultData,
			Timestamp:     completion.Timestamp,
		}

	case pb.MessageType_MESSAGE_TYPE_HEARTBEAT:
		// Respond to heartbeat
		log.Printf("💓 Received heartbeat message")
		return &pb.ConversationMessage{
			MessageId:     fmt.Sprintf("heartbeat-response-%d", time.Now().UnixNano()),
			CorrelationId: msg.CorrelationId,
			FromId:        a.config.AgentID,
			ToId:          "orchestrator",
			Type:          pb.MessageType_MESSAGE_TYPE_HEARTBEAT,
			Content:       "Agent is healthy",
			Timestamp:     timestamppb.Now(),
		}

	default:
		log.Printf("⚠️ Unknown message type: %v", msg.Type)
		return nil
	}
}

// startConversationStream opens and maintains a conversation stream with the orchestrator
func (a *AINativeAgent) startConversationStream(ctx context.Context) error {
	log.Printf("🔄 Starting conversation stream for agent %s", a.config.AgentID)

	// Open conversation stream
	stream, err := a.client.OpenConversation(ctx)
	if err != nil {
		return fmt.Errorf("failed to open conversation stream: %v", err)
	}

	// Send initial identification message
	identMsg := &pb.ConversationMessage{
		MessageId:     fmt.Sprintf("ident-%d", time.Now().UnixNano()),
		CorrelationId: "",
		FromId:        a.config.AgentID,
		ToId:          "orchestrator",
		Type:          pb.MessageType_MESSAGE_TYPE_STATUS_UPDATE,
		Content:       "Agent ready for instructions",
		Timestamp:     timestamppb.Now(),
	}

	if err := stream.Send(identMsg); err != nil {
		return fmt.Errorf("failed to send identification message: %v", err)
	}

	log.Printf("✅ Conversation stream established for agent %s", a.config.AgentID)

	// Listen for messages from orchestrator
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("🛑 Conversation stream context cancelled for agent %s", a.config.AgentID)
				return
			default:
				// Receive message from orchestrator
				msg, err := stream.Recv()
				if err != nil {
					log.Printf("❌ Error receiving message from stream: %v", err)
					return
				}

				log.Printf("📨 Received message from orchestrator: %s", msg.MessageId)

				// Process the message
				response := a.processConversationMessage(msg)
				if response != nil {
					// Send response back to orchestrator
					if err := stream.Send(response); err != nil {
						log.Printf("❌ Failed to send response: %v", err)
						return
					}
					log.Printf("📤 Sent response: %s", response.MessageId)
				}
			}
		}
	}()

	return nil
}
