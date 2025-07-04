# Conversation Graph Schema Implementation

## Overview
This document outlines the complete conversation graph schema for NeuroMesh, capturing all user interactions, AI decisions, and agent communications for continuity, learning, and auditability.

## Graph Schema Design

### Node Types

#### 1. Core Entity Nodes
```cypher
// User node representing system users  
(:User {
  id: string,              // Unique user identifier
  sessionId: string,       // Current session ID
  userType: string,        // web_session, api_user, agent, system
  status: string,          // active, inactive, blocked
  createdAt: datetime,
  updatedAt: datetime,
  lastSeen: datetime,
  metadata: map
})

// Session node for tracking user sessions
(:Session {
  id: string,              // Unique session identifier
  userId: string,          // Reference to user
  status: string,          // active, expired, closed
  createdAt: datetime,
  updatedAt: datetime,
  expiresAt: datetime,
  metadata: map
})

// Conversation node for multi-turn interactions
(:Conversation {
  id: string,              // Unique conversation identifier
  userId: string,          // User participating in conversation
  sessionId: string,       // Session context
  status: string,          // active, paused, closed, archived
  executionPlanIds: [string], // Linked execution plans
  createdAt: datetime,
  updatedAt: datetime
})

// Message node within conversations
(:ConversationMessage {
  id: string,              // Unique message identifier
  conversationId: string,  // Parent conversation
  role: string,            // user, assistant, system, agent
  content: string,         // Message content
  timestamp: datetime,
  metadata: map            // Additional context
})
```

#### 2. Request and Decision Nodes
```cypher
// User request node for tracking individual requests
(:UserRequest {
  id: string,              // Unique request identifier
  userId: string,          // User making the request
  sessionId: string,       // Session context
  conversationId: string,  // Optional conversation context
  userInput: string,       // Original user input
  analyzedIntent: string,  // AI-analyzed intent
  status: string,          // pending, analyzed, completed, failed
  createdAt: datetime
})

// AI analysis results
(:Analysis {
  id: string,              // Unique analysis identifier
  requestId: string,       // Associated request
  intent: string,          // Detected intent
  confidence: int,         // Confidence score (0-100)
  requiredAgents: [string], // List of required agent IDs
  complexity: string,      // simple, moderate, complex
  createdAt: datetime
})

// AI decision nodes
(:AIDecision {
  id: string,              // Unique decision identifier
  requestId: string,       // Associated request
  analysisId: string,      // Associated analysis
  type: string,            // clarify, execute
  reasoning: string,       // AI reasoning
  confidence: int,         // Decision confidence
  clarificationQuestion: string, // If type is clarify
  executionPlan: string,   // If type is execute
  createdAt: datetime
})
```

#### 3. Execution and Agent Nodes
```cypher
// Execution plan for agent coordination
(:ExecutionPlan {
  id: string,              // Unique plan identifier
  decisionId: string,      // Associated decision
  plan: string,            // Execution plan content
  status: string,          // pending, executing, completed, failed
  createdAt: datetime,
  startedAt: datetime,
  completedAt: datetime
})

// Agent nodes (already implemented)
(:Agent {
  id: string,              // Agent identifier
  name: string,            // Agent name
  description: string,     // Agent description
  status: string,          // healthy, busy, error, offline
  lastSeen: datetime,
  metadata: map
})

// Agent capabilities (already implemented)
(:AgentCapability {
  name: string,            // Capability name
  description: string,     // Capability description
  inputs: [string],        // Required inputs
  outputs: [string]        // Produced outputs
})
```

### Relationship Types

#### 1. Core Entity Relationships
```cypher
(:User)-[:HAS_SESSION]->(:Session)
(:User)-[:INITIATED]->(:UserRequest)
(:User)-[:PARTICIPATES_IN]->(:Conversation)
(:Session)-[:CONTAINS]->(:UserRequest)
(:Session)-[:INCLUDES]->(:Conversation)
```

#### 2. Conversation and Message Relationships
```cypher
(:Conversation)-[:CONTAINS_MESSAGE]->(:ConversationMessage)
(:Conversation)-[:LINKED_TO_PLAN]->(:ExecutionPlan)
(:UserRequest)-[:PART_OF]->(:Conversation)
(:UserRequest)-[:FOLLOWS]->(:UserRequest)  // Previous request relationship
```

#### 3. Analysis and Decision Relationships
```cypher
(:UserRequest)-[:ANALYZED_BY]->(:Analysis)
(:UserRequest)-[:RESULTED_IN]->(:AIDecision)
(:Analysis)-[:RESULTED_IN]->(:AIDecision)
(:AIDecision)-[:CREATED]->(:ExecutionPlan)
```

#### 4. Agent and Execution Relationships
```cypher
(:Agent)-[:HAS_CAPABILITY]->(:AgentCapability)
(:Agent)-[:EXECUTED]->(:ExecutionPlan)
(:ExecutionPlan)-[:REQUIRES_AGENT]->(:Agent)
(:ConversationMessage)-[:SENT_BY_AGENT]->(:Agent)  // For agent messages
```

#### 5. Temporal Flow Relationships
```cypher
(:UserRequest)-[:NEXT]->(:UserRequest)      // Request sequence
(:AIDecision)-[:NEXT]->(:AIDecision)        // Decision sequence
(:ConversationMessage)-[:NEXT]->(:ConversationMessage)  // Message sequence
```

## Integration Points

### 1. WebBFF Integration (PRIMARY)
**Location**: `/internal/web/bff.go`
**Integration Point**: `ProcessWebMessage` method

```go
func (w *WebBFF) ProcessWebMessage(ctx context.Context, sessionID, message string) (*WebResponse, error) {
    // 1. Create or get conversation for session
    conversation := w.getOrCreateConversation(sessionID)
    
    // 2. Add user message to conversation
    userMessage := conversation.AddMessage(generateMessageID(), domain.MessageRoleUser, message, nil)
    
    // 3. Process through orchestrator
    aiResponse := w.orchestrator.ProcessRequest(ctx, message, session.UserID)
    
    // 4. Add AI response to conversation
    assistantMessage := conversation.AddMessage(generateMessageID(), domain.MessageRoleAssistant, aiResponse.Message, nil)
    
    // 5. Link execution plan if created
    if aiResponse.ExecutionPlanID != "" {
        conversation.LinkExecutionPlan(aiResponse.ExecutionPlanID)
    }
    
    // 6. Persist conversation changes to graph
    w.conversationService.UpdateConversation(ctx, conversation)
}
```

### 2. Orchestrator Service Integration
**Location**: `/internal/orchestrator/application/orchestrator_service.go`
**Integration Point**: `ProcessUserRequest` method

```go
func (ors *OrchestratorService) ProcessUserRequest(ctx context.Context, request *OrchestratorRequest) (*OrchestratorResult, error) {
    // 1. Create UserRequest node
    userRequest := createUserRequest(request)
    
    // 2. Perform analysis and create Analysis node
    analysis := ors.aiDecisionEngine.ExploreAndAnalyze(ctx, request.UserInput, request.UserID, agentContext)
    
    // 3. Make decision and create AIDecision node
    decision := ors.aiDecisionEngine.MakeDecision(ctx, request.UserInput, request.UserID, analysis)
    
    // 4. Create ExecutionPlan if needed
    if decision.Type == DecisionTypeExecute {
        executionPlan := createExecutionPlan(decision)
        result.ExecutionPlanID = executionPlan.ID
    }
    
    // 5. Persist all relationships in graph
    ors.persistDecisionFlow(ctx, userRequest, analysis, decision, executionPlan)
}
```

### 3. Agent Communication Integration
**Location**: `/internal/grpc/server/orchestration_server.go`
**Integration Point**: `processIncomingMessage` method

```go
func (s *OrchestrationServer) processIncomingMessage(ctx context.Context, msg *pb.ConversationMessage) error {
    // 1. Find related conversation by correlation ID
    conversation := s.findConversationByCorrelation(msg.CorrelationId)
    
    // 2. Add agent message to conversation
    agentMessage := conversation.AddMessage(msg.MessageId, domain.MessageRoleAgent, msg.Content, convertContextToMetadata(msg.Context))
    
    // 3. Link message to agent
    s.linkMessageToAgent(agentMessage, msg.FromId)
    
    // 4. Persist to graph
    s.conversationService.UpdateConversation(ctx, conversation)
}
```

### 4. AI Message Bus Integration
**Location**: `/internal/messaging/ai_message_bus.go`
**Integration Point**: Message routing methods

```go
func (bus *AIMessageBusImpl) SendUserToAI(ctx context.Context, msg *UserToAIMessage) error {
    // 1. Create conversation message
    conversationMessage := createConversationMessage(msg)
    
    // 2. Store in conversation graph
    bus.storeMessageInConversation(ctx, conversationMessage, msg.CorrelationID)
    
    // 3. Route to AI
    return bus.messageBus.SendMessage(ctx, message)
}
```

## Schema Creation and Constraints

### Unique Constraints
```cypher
CREATE CONSTRAINT conversation_id_unique FOR (c:Conversation) REQUIRE c.id IS UNIQUE;
CREATE CONSTRAINT message_id_unique FOR (m:ConversationMessage) REQUIRE m.id IS UNIQUE;
CREATE CONSTRAINT user_request_id_unique FOR (ur:UserRequest) REQUIRE ur.id IS UNIQUE;
CREATE CONSTRAINT analysis_id_unique FOR (a:Analysis) REQUIRE a.id IS UNIQUE;
CREATE CONSTRAINT ai_decision_id_unique FOR (ad:AIDecision) REQUIRE ad.id IS UNIQUE;
CREATE CONSTRAINT execution_plan_id_unique FOR (ep:ExecutionPlan) REQUIRE ep.id IS UNIQUE;
```

### Performance Indexes
```cypher
CREATE INDEX conversation_user_idx FOR (c:Conversation) ON (c.userId);
CREATE INDEX conversation_session_idx FOR (c:Conversation) ON (c.sessionId);
CREATE INDEX conversation_status_idx FOR (c:Conversation) ON (c.status);
CREATE INDEX message_conversation_idx FOR (m:ConversationMessage) ON (m.conversationId);
CREATE INDEX message_role_idx FOR (m:ConversationMessage) ON (m.role);
CREATE INDEX message_timestamp_idx FOR (m:ConversationMessage) ON (m.timestamp);
CREATE INDEX user_request_user_idx FOR (ur:UserRequest) ON (ur.userId);
CREATE INDEX user_request_session_idx FOR (ur:UserRequest) ON (ur.sessionId);
CREATE INDEX user_request_status_idx FOR (ur:UserRequest) ON (ur.status);
```

## Implementation Priority

1. **Phase 1**: Conversation domain and infrastructure (✅ COMPLETED)
2. **Phase 2**: WebBFF integration for user message persistence
3. **Phase 3**: Orchestrator service integration for decision tracking
4. **Phase 4**: Agent communication integration for agent message tracking
5. **Phase 5**: Learning and pattern analysis features

## Benefits

1. **Continuity**: Full conversation history for context-aware AI responses
2. **Learning**: Pattern analysis for improving AI decision making
3. **Auditability**: Complete trace of all interactions and decisions
4. **Analytics**: Rich data for system performance and user behavior analysis
5. **Debugging**: Full conversation flow for troubleshooting issues
