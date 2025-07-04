// AI-Native Text Processing Agent
// Clean architecture implementation with proper separation of concerns
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ztdp/agents/text-processor/agent"
)

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	log.Printf("🚀 Starting AI-Native Text Processing Agent...")

	// Configuration
	config := agent.Config{
		AgentID:             getEnv("AGENT_ID", "text-processor-001"),
		Name:                "AI-Native Text Processing Agent",
		OrchestratorAddress: getEnv("ORCHESTRATOR_ADDRESS", "localhost:50051"),
		ReconnectInterval:   30 * time.Second,
	}

	// Create the AI-native agent
	textAgent := agent.NewAINativeAgent(config)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the agent (includes registration, infrastructure, and AI conversation stream)
	if err := textAgent.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start agent: %v", err)
	}

	// Agent is now running with:
	// ✅ Registration complete
	// ✅ Dedicated heartbeat process (30s intervals)
	// ✅ Dedicated status monitoring process
	// ✅ AI conversation stream (for instructions/completions)

	log.Printf("🎯 Agent %s ready for AI instructions!", config.AgentID)
	log.Printf("🔗 Connected to orchestrator at %s", config.OrchestratorAddress)
	log.Printf("🤖 Capabilities: word-count, text-analysis, character-count")

	// Wait for interrupt signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal
	<-sigChan
	log.Printf("🛑 Received shutdown signal, stopping agent...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := textAgent.Stop(shutdownCtx); err != nil {
		log.Printf("⚠️ Error during shutdown: %v", err)
	}

	log.Printf("✅ AI-Native Text Processing Agent stopped gracefully")
}
