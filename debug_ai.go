package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"neuromesh/internal/ai/infrastructure"
	"neuromesh/internal/logging"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	config := &infrastructure.OpenAIConfig{APIKey: apiKey}
	logger := logging.NewNoOpLogger()
	aiProvider := infrastructure.NewOpenAIProvider(config, logger)
	
	agentContext := `Available agents:
- text-processor (ID: text-processor, Status: online)
  Capabilities: word-count, text-analysis`

	systemPrompt := fmt.Sprintf(`You are an AI orchestrator that coordinates with specialized agents to help users.

Available agents:
%s

Your capabilities:
1. Analyze user requests and determine which agents can help
2. Send events to agents with specific tasks
3. Process agent responses and provide final answers to users

When you want to send an event to an agent, use this EXACT format:
SEND_EVENT:
Agent: [agent-id]
Action: [what you want the agent to do]
Content: [the specific content/data for the agent]
Intent: [brief description of what you're trying to achieve]

When you have a final response for the user, use this EXACT format:
USER_RESPONSE: [your response to the user]

Always be helpful, accurate, and conversational in your responses.`, agentContext)

	userPrompt := "User request: Count the words in this text: Hello world testing"

	response, err := aiProvider.CallAI(context.Background(), systemPrompt, userPrompt)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("AI Response:\n%s\n", response)
	fmt.Printf("\nContains SEND_EVENT: %t\n", strings.Contains(response, "SEND_EVENT"))
}
