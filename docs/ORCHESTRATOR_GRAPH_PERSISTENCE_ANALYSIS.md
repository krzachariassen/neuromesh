# Orchestrator Graph Persistence Analysis

## End-to-End Flow Analysis

This document analyzes the complete orchestrator flow from `ProcessUserRequest` to identify all data that needs to be persisted in the graph and how it should be linked to existing conversation, user, and agent nodes.

## Current Flow Trace

### 1. Entry Point: `ProcessUserRequest`
Location: `/internal/orchestrator/application/orchestrator_service.go`

```go
func (ors *OrchestratorService) ProcessUserRequest(ctx context.Context, request domain.UserRequest) (*domain.UserResponse, error)
```

**Input Data:**
- `domain.UserRequest` with:
  - `UserID` (links to User node)
  - `UserInput` (the actual message)
  - `ConversationID` (links to Conversation node)

### 2. Agent Context Discovery
```go
agentContext, err := ors.graphExplorer.GetAgentContext(ctx, request.UserID)
```

**Generated Data:**
- Agent discovery query results
- Available agent capabilities and metadata
- **PERSISTENCE NEED:** Link user request to discovered agents

### 3. Planning Domain: AI Analysis
```go
analysis, err := ors.aiDecisionEngine.ExploreAndAnalyze(ctx, request.UserInput, request.UserID, agentContext)
```

**Generated Data:**
- `domain.Analysis` object with:
  - `ID`, `RequestID`, `Intent`, `Category`
  - `Confidence` (0-100)
  - `RequiredAgents` (list of agent IDs)
  - `Reasoning` (AI explanation)
  - `Timestamp`

**PERSISTENCE NEED:** 
- Create Analysis node linked to:
  - User node (via UserID)
  - Conversation node (via request context)
  - Agent nodes (via RequiredAgents)
  - Store all AI reasoning and confidence data

### 4. Decision Domain: AI Decision Making
```go
decision, err := ors.aiDecisionEngine.MakeDecision(ctx, request.UserInput, request.UserID, analysis)
```

**Generated Data:**
- `domain.Decision` object with:
  - `ID`, `RequestID`, `AnalysisID`
  - `Type` (CLARIFY|EXECUTE)
  - `ClarificationQuestion` (if clarifying)
  - `ExecutionPlan` (if executing)
  - `AgentCoordination` (if executing)
  - `Reasoning`
  - `Timestamp`

**PERSISTENCE NEED:**
- Create Decision node linked to:
  - Analysis node (via AnalysisID)
  - User node
  - Conversation node
  - Store decision type, reasoning, and any execution plans

### 5A. Clarification Flow (if decision.Type == CLARIFY)
- Returns clarification question to user
- **PERSISTENCE NEED:** Track that clarification was requested

### 5B. Execution Domain: AI Execution Engine
```go
executionResult, err := ors.aiExecutionEngine.ExecuteWithAgents(ctx, decision.ExecutionPlan, request.UserInput, request.UserID, agentContext)
```

**Generated Data:**
- `ExecutionPlan` domain objects with:
  - `ID`, `Action`, `Parameters`, `Steps[]`
  - `Status` (PENDING → IN_PROGRESS → COMPLETED/FAILED)
  - `CreatedAt`, `StartedAt`, `CompletedAt`
  - `Error` (if failed)
- Individual `ExecutionStep` objects for each agent interaction
- Agent message correlation tracking
- Final execution results

**PERSISTENCE NEED:**
- Create ExecutionPlan node linked to:
  - Decision node
  - User node
  - Conversation node
- Create ExecutionStep nodes linked to:
  - ExecutionPlan node
  - Agent nodes (via AgentID)
- Track all status changes and timing
- Store final execution results

## Proposed Graph Schema Extensions

### New Node Types

#### 1. Analysis Node
```cypher
CREATE (a:Analysis {
  id: String,
  request_id: String,
  intent: String,
  category: String,
  confidence: Integer,
  required_agents: [String],
  reasoning: String,
  timestamp: DateTime
})
```

**Relationships:**
- `(User)-[:REQUESTED_ANALYSIS]->(Analysis)`
- `(Conversation)-[:CONTAINS_ANALYSIS]->(Analysis)`
- `(Analysis)-[:REQUIRES_AGENT]->(Agent)`
- `(Analysis)-[:LEADS_TO]->(Decision)`

#### 2. Decision Node
```cypher
CREATE (d:Decision {
  id: String,
  request_id: String,
  analysis_id: String,
  type: String, // "CLARIFY" or "EXECUTE"
  clarification_question: String,
  execution_plan: String,
  agent_coordination: String,
  reasoning: String,
  timestamp: DateTime
})
```

**Relationships:**
- `(Analysis)-[:LEADS_TO]->(Decision)`
- `(User)-[:MADE_DECISION]->(Decision)`
- `(Conversation)-[:CONTAINS_DECISION]->(Decision)`
- `(Decision)-[:CREATES]->(ExecutionPlan)` (if type=EXECUTE)

#### 3. ExecutionPlan Node
```cypher
CREATE (ep:ExecutionPlan {
  id: String,
  action: String,
  parameters: Map,
  status: String,
  created_at: DateTime,
  started_at: DateTime,
  completed_at: DateTime,
  error: String
})
```

**Relationships:**
- `(Decision)-[:CREATES]->(ExecutionPlan)`
- `(User)-[:EXECUTES]->(ExecutionPlan)`
- `(Conversation)-[:CONTAINS_EXECUTION]->(ExecutionPlan)`
- `(ExecutionPlan)-[:HAS_STEP]->(ExecutionStep)`

#### 4. ExecutionStep Node
```cypher
CREATE (es:ExecutionStep {
  id: String,
  name: String,
  agent_id: String,
  action: String,
  parameters: Map,
  status: String,
  depends_on: [String],
  created_at: DateTime,
  started_at: DateTime,
  completed_at: DateTime,
  error: String
})
```

**Relationships:**
- `(ExecutionPlan)-[:HAS_STEP]->(ExecutionStep)`
- `(ExecutionStep)-[:EXECUTED_BY]->(Agent)`
- `(ExecutionStep)-[:DEPENDS_ON]->(ExecutionStep)`

## Implementation Plan

### Phase 1: Analysis Domain Graph Persistence
1. **RED:** Write failing tests for Analysis graph persistence
2. **GREEN:** Implement Analysis repository and domain changes
3. **REFACTOR:** Clean up Analysis creation in planning domain
4. **VALIDATE:** Ensure all tests pass

### Phase 2: Decision Domain Graph Persistence
1. **RED:** Write failing tests for Decision graph persistence
2. **GREEN:** Implement Decision repository and domain changes
3. **REFACTOR:** Clean up Decision creation in planning domain
4. **VALIDATE:** Ensure all tests pass

### Phase 3: Execution Domain Graph Persistence
1. **RED:** Write failing tests for ExecutionPlan/Step graph persistence
2. **GREEN:** Implement Execution repository and domain changes
3. **REFACTOR:** Clean up Execution creation in execution domain
4. **VALIDATE:** Ensure all tests pass

### Phase 4: Integration Testing
1. **RED:** Write end-to-end integration tests
2. **GREEN:** Ensure full orchestrator flow persists correctly
3. **REFACTOR:** Optimize graph queries and relationships
4. **VALIDATE:** Full system testing

## Data Flow Summary

```
User Request → Agent Discovery → AI Analysis → AI Decision → Execution
     ↓             ↓               ↓           ↓           ↓
   User Node → Agent Links → Analysis Node → Decision Node → ExecutionPlan Node
                                                                ↓
                                                          ExecutionStep Nodes
                                                                ↓
                                                            Agent Links
```

## Next Steps

1. Start with Analysis domain graph persistence (highest value, cleanest implementation)
2. Review implementation approach with user before proceeding
3. Follow TDD strictly for each domain
4. Ensure all relationships link properly to existing Conversation/User/Agent nodes
