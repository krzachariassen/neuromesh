# AI-Native Execution Engine Design

## 🎯 Core Principle
**AI must be in the loop of EVERY execution step with bidirectional communication**

## 🚫 What We Had (Wrong)
```
User: "Count words: This is a tree"
AI: Creates execution plan
System: Executes plan mechanically  
User: Gets result
```

## ✅ What We Need (AI-Native)
```
User: "Count words: This is a tree"
AI: "I'll use text-processor. Let me ask it to count words."
AI → text-processor: "Count words in 'This is a tree'"
text-processor → AI: "Result: 5 words"
AI: "Perfect! I'll format this for the user."
AI → User: "The text contains 5 words"
```

## 🏗️ Simple Architecture

### 1. AI Conversation Engine (Core)
```go
type AIConversationEngine struct {
    aiProvider AIProvider
    messageBus MessageBus  // For agent communication
}

func (e *AIConversationEngine) ProcessWithAgents(ctx context.Context, userInput string, userID string) (string, error)
```

### 2. Message-Based Agent Communication
```go
type AgentMessage struct {
    From        string                 `json:"from"`        // "ai-orchestrator" 
    To          string                 `json:"to"`          // "text-processor"
    Content     string                 `json:"content"`     // "Count words in 'This is a tree'"
    Type        string                 `json:"type"`        // "request" | "response" | "question"
    Parameters  map[string]interface{} `json:"parameters"`
    ConversationID string              `json:"conversation_id"`
}
```

### 3. AI Orchestrates Every Step
```go
// AI decides what to do next after each agent response
func (e *AIConversationEngine) ProcessAgentResponse(ctx context.Context, agentResponse *AgentMessage) (*NextAction, error) {
    // AI analyzes agent response and decides:
    // - Call another agent?
    // - Ask agent for clarification? 
    // - Respond to user?
    // - Need more information?
}
```

## 🔄 Execution Flow

1. **User Input** → AI Orchestrator
2. **AI Analyzes** → Decides to call text-processor
3. **AI → Agent**: "Count words in 'This is a tree'"
4. **Agent → AI**: "5 words"
5. **AI Analyzes Response** → Decides it's complete
6. **AI → User**: "The text contains 5 words"

## 🎪 Key Benefits

- **AI Mediates Everything**: No rigid execution plans
- **Bidirectional**: Agents can ask AI questions
- **Adaptive**: AI can change course based on agent responses
- **Simple**: Message-based, no complex orchestration
- **Conversational**: Natural interaction between AI and agents

## 🛠️ Implementation Plan

### Phase 1: AI Conversation Engine (TDD)
- RED: Test AI processing user input and calling agent
- GREEN: Implement basic AI → Agent → AI flow
- REFACTOR: Clean up interfaces

### Phase 2: Agent Response Processing (TDD)  
- RED: Test AI processing agent responses
- GREEN: Implement AI decision making on agent responses
- REFACTOR: Optimize conversation flow

### Phase 3: Multi-Agent Coordination (TDD)
- RED: Test AI coordinating multiple agents
- GREEN: Implement agent-to-agent via AI mediation
- REFACTOR: Perfect the conversation flow

## 🚀 Example Conversation

```
User: "Count words: This is a tree"

AI Internal: "User wants word count. I have text-processor with word-count capability."

AI → text-processor: {
  "content": "Count words in the text: This is a tree",
  "type": "request",
  "parameters": {"text": "This is a tree", "action": "count-words"}
}

text-processor → AI: {
  "content": "Word count completed",
  "type": "response", 
  "parameters": {"word_count": 5, "text": "This is a tree"}
}

AI Internal: "Perfect! text-processor counted 5 words. I can respond to user."

AI → User: "I analyzed your text using the text-processor agent. The text 'This is a tree' contains 5 words."
```

## 🎯 Next Steps

1. Replace ExecutionCoordinator with AIConversationEngine
2. Use existing MessageBus for agent communication
3. Let AI decide every step dynamically
4. Keep it simple - no complex orchestration logic
