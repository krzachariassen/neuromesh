// Package agent provides an AI-native text translation agent implementation
package agent

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/ztdp/agents/text-translator/proto/api"
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

	// Start dedicated infrastructure processes (heartbeat, status)
	if err := a.StartInfrastructure(ctx); err != nil {
		return fmt.Errorf("failed to start infrastructure: %w", err)
	}

	// Start AI conversation stream (separate from infrastructure)
	if err := a.startConversationStream(ctx); err != nil {
		return fmt.Errorf("failed to start AI conversation stream: %w", err)
	}

	log.Printf("✅ AI-native text translation agent started successfully")
	log.Printf("🎯 Agent %s ready for AI instructions!", a.config.AgentID)
	log.Printf("🔗 Connected to orchestrator at %s", a.config.OrchestratorAddress)
	log.Printf("🌐 Capabilities: translate-text, detect-language, format-translation")
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
		Type:         "text-translator",
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
			Name:        "translate-text",
			Description: "Translate text between different languages",
			Inputs:      []string{"text", "target_language"},
			Outputs:     []string{"translated_text"},
		},
		{
			Name:        "detect-language",
			Description: "Detect the language of input text",
			Inputs:      []string{"text"},
			Outputs:     []string{"detected_language"},
		},
		{
			Name:        "format-translation",
			Description: "Format translation results with metadata",
			Inputs:      []string{"text", "source_language", "target_language"},
			Outputs:     []string{"formatted_translation"},
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

	if strings.Contains(instructionLower, "translate") {
		targetLang := a.extractTargetLanguage(instruction)
		translation := a.translateText(text, targetLang)
		response := fmt.Sprintf(`Translation to %s: "%s"`, targetLang, translation)
		log.Printf("✅ Response: %s", response)
		return response
	}

	if strings.Contains(instructionLower, "detect") && strings.Contains(instructionLower, "language") {
		language := a.detectLanguage(text)
		response := fmt.Sprintf(`Detected language: %s`, language)
		log.Printf("✅ Response: %s", response)
		return response
	}

	if strings.Contains(instructionLower, "format") && strings.Contains(instructionLower, "translation") {
		targetLang := a.extractTargetLanguage(instruction)
		formatted := a.formatTranslation(text, "auto", targetLang)
		response := fmt.Sprintf(`Formatted translation: %s`, formatted)
		log.Printf("✅ Response: %s", response)
		return response
	}

	// Default: translate to English (most common request)
	translation := a.translateText(text, "English")
	response := fmt.Sprintf(`Translation to English: "%s"`, translation)
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

// translateText translates text to the target language (simulated)
func (a *AINativeAgent) translateText(text, targetLang string) string {
	if text == "" {
		return ""
	}

	// Simulated translation logic - in real implementation, this would call translation API
	translations := map[string]map[string]string{
		"hello world": {
			"Spanish":    "Hola mundo",
			"French":     "Bonjour le monde",
			"German":     "Hallo Welt",
			"Italian":    "Ciao mondo",
			"Portuguese": "Olá mundo",
		},
		"the quick brown fox": {
			"Spanish":    "el zorro marrón rápido",
			"French":     "le renard brun rapide",
			"German":     "der schnelle braune Fuchs",
			"Italian":    "la volpe marrone veloce",
			"Portuguese": "a raposa marrom rápida",
		},
	}

	textLower := strings.ToLower(text)
	if langMap, exists := translations[textLower]; exists {
		if translation, hasLang := langMap[targetLang]; hasLang {
			return translation
		}
	}

	// Fallback: simulate translation by adding language prefix
	return fmt.Sprintf("[%s] %s", targetLang, text)
}

// detectLanguage detects the language of input text (simulated)
func (a *AINativeAgent) detectLanguage(text string) string {
	if text == "" {
		return "unknown"
	}

	// Simple language detection simulation
	textLower := strings.ToLower(text)

	if strings.Contains(textLower, "hola") || strings.Contains(textLower, "mundo") {
		return "Spanish"
	}
	if strings.Contains(textLower, "bonjour") || strings.Contains(textLower, "monde") {
		return "French"
	}
	if strings.Contains(textLower, "hallo") || strings.Contains(textLower, "welt") {
		return "German"
	}
	if strings.Contains(textLower, "ciao") {
		return "Italian"
	}

	return "English" // Default assumption
}

// formatTranslation formats translation results with metadata
func (a *AINativeAgent) formatTranslation(text, sourceLang, targetLang string) string {
	if sourceLang == "auto" {
		sourceLang = a.detectLanguage(text)
	}

	translation := a.translateText(text, targetLang)

	return fmt.Sprintf("Source: %s\nTarget: %s\nOriginal: \"%s\"\nTranslation: \"%s\"",
		sourceLang, targetLang, text, translation)
}

// extractTargetLanguage extracts target language from instruction
func (a *AINativeAgent) extractTargetLanguage(instruction string) string {
	instructionLower := strings.ToLower(instruction)

	languages := []string{"spanish", "french", "german", "italian", "portuguese", "english"}
	for _, lang := range languages {
		if strings.Contains(instructionLower, lang) {
			return strings.Title(lang)
		}
	}

	// Look for "to X" pattern
	re := regexp.MustCompile(`to\s+(\w+)`)
	matches := re.FindStringSubmatch(instructionLower)
	if len(matches) > 1 {
		return strings.Title(matches[1])
	}

	return "English" // Default
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

// Legacy heartbeat methods - DEPRECATED in favor of dedicated infrastructure processes
// StartHeartbeat - DEPRECATED: Use StartInfrastructure() instead
func (a *AINativeAgent) StartHeartbeat(ctx context.Context, notificationChan chan<- bool) error {
	log.Printf("⚠️ DEPRECATED: StartHeartbeat called - use StartInfrastructure() instead")
	// For backward compatibility, start the infrastructure
	return a.StartInfrastructure(ctx)
}

// Legacy heartbeat methods - REMOVED in favor of dedicated infrastructure processes

// processConversationMessage handles ONLY AI conversation messages (instructions/completions)
func (a *AINativeAgent) processConversationMessage(msg *pb.ConversationMessage) *pb.ConversationMessage {
	log.Printf("📨 Processing AI conversation message: %s (type: %v)", msg.MessageId, msg.Type)

	switch msg.Type {
	case pb.MessageType_MESSAGE_TYPE_INSTRUCTION:
		// Process the AI instruction and create a completion response
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

	default:
		log.Printf("⚠️ Unexpected message type in conversation stream: %v (infrastructure messages should use dedicated endpoints)", msg.Type)
		return nil
	}
}

// startConversationStream opens and maintains a PURE AI conversation stream
func (a *AINativeAgent) startConversationStream(ctx context.Context) error {
	log.Printf("🔄 Starting AI conversation stream for agent %s", a.config.AgentID)

	// Create context with agent ID in metadata (no identification message needed!)
	md := metadata.New(map[string]string{
		"agent-id": a.config.AgentID,
	})
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	// Open conversation stream with agent ID in metadata
	stream, err := a.client.OpenConversation(streamCtx)
	if err != nil {
		return fmt.Errorf("failed to open conversation stream: %v", err)
	}

	log.Printf("✅ AI conversation stream established for agent %s", a.config.AgentID)

	// Listen ONLY for AI instruction messages (no identification message needed)
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("🛑 AI conversation stream context cancelled for agent %s", a.config.AgentID)
				return
			default:
				// Receive AI instruction from orchestrator
				msg, err := stream.Recv()
				if err != nil {
					log.Printf("❌ Error receiving AI message from stream: %v", err)
					return
				}

				log.Printf("🧠 Received AI instruction: %s", msg.MessageId)

				// Process the AI instruction
				response := a.processConversationMessage(msg)
				if response != nil {
					// Send completion response back to AI
					if err := stream.Send(response); err != nil {
						log.Printf("❌ Failed to send AI response: %v", err)
						return
					}
					log.Printf("🧠 Sent AI completion: %s", response.MessageId)
				}
			}
		}
	}()

	return nil
}

// StartInfrastructure starts all dedicated infrastructure processes
func (a *AINativeAgent) StartInfrastructure(ctx context.Context) error {
	log.Printf("🔧 Starting infrastructure processes for agent %s", a.config.AgentID)

	// Start heartbeat process
	if err := a.startHeartbeatProcess(ctx); err != nil {
		return fmt.Errorf("failed to start heartbeat process: %w", err)
	}

	// Start status monitoring process
	if err := a.startStatusProcess(ctx); err != nil {
		return fmt.Errorf("failed to start status process: %w", err)
	}

	log.Printf("✅ All infrastructure processes started for agent %s", a.config.AgentID)
	return nil
}

// startHeartbeatProcess starts a dedicated heartbeat process using the dedicated endpoint
func (a *AINativeAgent) startHeartbeatProcess(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		log.Printf("💓 Starting dedicated heartbeat process for agent %s", a.config.AgentID)

		// Send immediate first heartbeat
		a.sendInfrastructureHeartbeat(ctx)

		for {
			select {
			case <-ticker.C:
				a.sendInfrastructureHeartbeat(ctx)
			case <-ctx.Done():
				log.Printf("💓 Heartbeat process stopped for agent %s", a.config.AgentID)
				return
			}
		}
	}()

	return nil
}

// startStatusProcess starts a dedicated status update process
func (a *AINativeAgent) startStatusProcess(ctx context.Context) error {
	go func() {
		log.Printf("🔧 Starting dedicated status process for agent %s", a.config.AgentID)

		// Send initial status
		a.sendStatusUpdate(ctx, pb.AgentStatus_AGENT_STATUS_HEALTHY)

		// Listen for status changes (for now, just healthy)
		// In the future, this could monitor agent health and send updates
		<-ctx.Done()
		log.Printf("🔧 Status process stopped for agent %s", a.config.AgentID)
	}()

	return nil
}

// sendInfrastructureHeartbeat sends heartbeat using dedicated Heartbeat endpoint
func (a *AINativeAgent) sendInfrastructureHeartbeat(ctx context.Context) {
	if a.client != nil {
		heartbeatReq := &pb.HeartbeatRequest{
			AgentId:   a.config.AgentID,
			SessionId: a.sessionID,
			Status:    pb.AgentStatus_AGENT_STATUS_HEALTHY,
		}

		_, err := a.client.Heartbeat(ctx, heartbeatReq)
		if err != nil {
			log.Printf("❌ Infrastructure heartbeat failed for agent %s: %v", a.config.AgentID, err)
			return
		}

		log.Printf("💓 Infrastructure heartbeat sent for agent %s", a.config.AgentID)
	}
}

// sendStatusUpdate sends status using dedicated UpdateAgentStatus endpoint
func (a *AINativeAgent) sendStatusUpdate(ctx context.Context, status pb.AgentStatus) {
	if a.client != nil {
		statusReq := &pb.UpdateAgentStatusRequest{
			AgentId:   a.config.AgentID,
			SessionId: a.sessionID,
			Status:    status,
			Timestamp: timestamppb.Now(),
		}

		_, err := a.client.UpdateAgentStatus(ctx, statusReq)
		if err != nil {
			log.Printf("❌ Status update failed for agent %s: %v", a.config.AgentID, err)
			return
		}

		log.Printf("🔧 Status update sent for agent %s: %v", a.config.AgentID, status)
	}
}
