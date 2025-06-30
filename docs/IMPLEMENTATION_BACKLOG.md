# ZTDP Implementation Backlog

## 🏆 COMPLETED PHASES

### ✅ Phase 1: AI-Native Orchestration Foundation  
**Status: 100% COMPLETE**
- Real AI decision making with OpenAI GPT-4
- Event-driven architecture with RabbitMQ
- Bidirectional conversations (AI ↔ Agent ↔ AI ↔ User)
- Zero simulation code - pure event-driven architecture
- TDD validated: All 13 tests GREEN with real AI behavior
- Removed MockAIProvider entirely per user requirements

### ✅ Phase 1.5: Agent Architecture Modernization
**Status: 100% COMPLETE**  
- Refactored text-processor agent to clean architecture
- Updated protobuf spec to AI-native conversational methods
- Removed all demo files and legacy code
- Synchronized orchestrator and agent protobuf specifications
- All builds and tests pass cleanly

---

## 🔥 PHASE 2: INFRASTRUCTURE COMPLETION 

### Priority 1: Agent Health Monitoring (URGENT)

#### 2.1 Agent Heartbeat Implementation
**TDD Steps:**
1. **RED**: Write test for agent heartbeat sending
2. **GREEN**: Implement heartbeat in agent
3. **REFACTOR**: Clean up heartbeat logic
4. **VALIDATE**: Test with real orchestrator

```go
// agents/text-processor/agent/agent.go
func (a *Agent) StartHeartbeat() {
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for range ticker.C {
            req := &pb.HeartbeatRequest{
                AgentId: a.agentID,
                SessionId: a.sessionID,
                Status: pb.AgentStatus_AGENT_STATUS_HEALTHY,
                HealthMetrics: a.getHealthMetrics(),
            }
            _, err := a.grpcClient.Heartbeat(context.Background(), req)
            if err != nil {
                a.logger.Error("Heartbeat failed", "error", err)
            }
        }
    }()
}

func (a *Agent) getHealthMetrics() *structpb.Struct {
    return &structpb.Struct{
        Fields: map[string]*structpb.Value{
            "uptime": structpb.NewNumberValue(time.Since(a.startTime).Seconds()),
            "processed_messages": structpb.NewNumberValue(float64(a.processedCount)),
            "memory_usage": structpb.NewNumberValue(a.getMemoryUsage()),
        },
    }
}
```

#### 2.2 Registry Health Management  
**TDD Steps:**
1. **RED**: Write test for agent timeout detection
2. **GREEN**: Implement timeout monitoring in registry
3. **REFACTOR**: Clean up monitoring logic
4. **VALIDATE**: Test agent disconnect scenarios

```go
// internal/agentRegistry/registry.go
func (r *Registry) StartHealthMonitoring() {
    ticker := time.NewTicker(10 * time.Second) // Check every 10s
    go func() {
        for range ticker.C {
            r.checkAgentHealth()
        }
    }()
}

func (r *Registry) checkAgentHealth() {
    threshold := time.Now().Add(-30 * time.Second)
    
    r.agents.Range(func(key, value interface{}) bool {
        agent := value.(*domain.Agent)
        if agent.LastHeartbeat.Before(threshold) && agent.Status != domain.AgentStatusDisconnected {
            r.markAgentDisconnected(agent.ID, "Heartbeat timeout")
        }
        return true
    })
}
```

#### 2.3 Graph Cleanup Implementation
**TDD Steps:**
1. **RED**: Write test for stale agent removal
2. **GREEN**: Implement graph cleanup on disconnect
3. **REFACTOR**: Optimize cleanup queries  
4. **VALIDATE**: Test with Neo4j graph

```go
// internal/graph/neo4j_graph.go
func (g *Neo4jGraph) RemoveStaleAgent(ctx context.Context, agentID string) error {
    query := `
        MATCH (a:Agent {id: $agentId})
        DETACH DELETE a
    `
    return g.executeQuery(ctx, query, map[string]interface{}{
        "agentId": agentID,
    })
}

func (g *Neo4jGraph) CleanupTestArtifacts(ctx context.Context) error {
    // Remove agents created during testing
    query := `
        MATCH (a:Agent)
        WHERE a.created_by = "test" OR a.name CONTAINS "test"
        DETACH DELETE a
    `
    return g.executeQuery(ctx, query, nil)
}
```

### Priority 2: Orchestrator gRPC Server Updates

#### 2.4 Update Server for New Protobuf Spec
**Current Issue**: Need to verify orchestrator gRPC server matches new AI-native protobuf methods

**Investigation Required:**
- Check if all protobuf methods are implemented
- Verify method signatures match new spec
- Test gRPC streaming functionality
- Update any legacy method implementations

```bash
# Check current server implementation
cd /mnt/c/Work/git/ztdp/orchestrator
go build ./... # Should pass cleanly
go test ./internal/grpc/server/... # Check for any failures
```

---

## 🎨 PHASE 2.5: USER EXPERIENCE ENHANCEMENT

### Priority 3: UI Modernization

#### 2.5.1 Modern Chat Interface
**Current**: Basic HTML/JS in `/static/chat.html`
**Target**: Modern, responsive interface with:
- Real-time gRPC streaming display
- Agent status indicators  
- Conversation history
- Multi-agent orchestration visualization

**TDD Steps:**
1. **RED**: Write tests for streaming message display
2. **GREEN**: Implement WebSocket/gRPC-web integration  
3. **REFACTOR**: Clean up UI components
4. **VALIDATE**: Test with real conversations

#### 2.5.2 Agent Status Dashboard
```javascript
// static/js/agent-dashboard.js
class AgentDashboard {
    constructor() {
        this.agents = new Map();
        this.startStatusUpdates();
    }
    
    startStatusUpdates() {
        // WebSocket connection for real-time agent status
        this.ws = new WebSocket('ws://localhost:8080/agents/status');
        this.ws.onmessage = (event) => {
            const update = JSON.parse(event.data);
            this.updateAgentStatus(update);
        };
    }
    
    updateAgentStatus(update) {
        const indicator = document.getElementById(`agent-${update.agentId}`);
        indicator.className = `status-${update.status.toLowerCase()}`;
        indicator.textContent = update.status;
    }
}
```

### Priority 4: End-to-End Testing

#### 2.5.3 Full Browser Testing
**TDD Steps:**  
1. **RED**: Write browser automation tests
2. **GREEN**: Implement full user journey test
3. **REFACTOR**: Clean up test automation
4. **VALIDATE**: Test in multiple browsers

```go
// test/e2e/browser_test.go
func TestFullUserJourney_Browser(t *testing.T) {
    // Start orchestrator and agent
    ctx := context.Background()
    
    // Open browser to chat interface
    driver := setupWebDriver(t)
    defer driver.Quit()
    
    // Type message: "Count words: Hello world"
    driver.FindElement(selenium.ByID, "message-input").SendKeys("Count words: Hello world")
    driver.FindElement(selenium.ByID, "send-button").Click()
    
    // Wait for AI response
    response := waitForResponse(driver, 10*time.Second)
    assert.Contains(t, response, "2 words")
    
    // Verify agent status shown as active
    agentStatus := driver.FindElement(selenium.ByID, "agent-text-processor-status").GetText()
    assert.Equal(t, "HEALTHY", agentStatus)
}
```

---

## 🚀 PHASE 3: MULTI-AGENT ORCHESTRATION

### Priority 5: Conversation Plans

#### 3.1 ConversationPlan Data Structure
**TDD Steps:**
1. **RED**: Write tests for plan creation and storage
2. **GREEN**: Implement ConversationPlan struct and methods
3. **REFACTOR**: Optimize plan execution logic
4. **VALIDATE**: Test with real multi-agent scenarios

```go
// internal/orchestrator/domain/conversation_plan.go
type ConversationPlan struct {
    ID               string                 `json:"id"`
    UserID           string                 `json:"user_id"`
    UserRequest      string                 `json:"user_request"`
    AIStrategy       string                 `json:"ai_strategy"`
    ExecutionPattern ExecutionPattern       `json:"execution_pattern"`
    Steps            []ConversationStep     `json:"steps"`
    Dependencies     map[string][]string    `json:"dependencies"`
    Status           PlanStatus            `json:"status"`
    CreatedAt        time.Time             `json:"created_at"`
    UpdatedAt        time.Time             `json:"updated_at"`
    CompletedAt      *time.Time            `json:"completed_at,omitempty"`
}

type ConversationStep struct {
    ID           string           `json:"id"`
    PlanID       string           `json:"plan_id"`
    AgentID      string           `json:"agent_id"`
    Capability   string           `json:"capability"`
    Instruction  string           `json:"instruction"`
    Dependencies []string         `json:"dependencies"`
    Status       StepStatus       `json:"status"`
    Response     string           `json:"response,omitempty"`
    ErrorMessage string           `json:"error_message,omitempty"`
    StartedAt    *time.Time       `json:"started_at,omitempty"`
    CompletedAt  *time.Time       `json:"completed_at,omitempty"`
}

type ExecutionPattern string
const (
    ExecutionPatternSequential  ExecutionPattern = "sequential"
    ExecutionPatternParallel    ExecutionPattern = "parallel"
    ExecutionPatternConditional ExecutionPattern = "conditional"
    ExecutionPatternIterative   ExecutionPattern = "iterative"
)

type PlanStatus string
const (
    PlanStatusCreated    PlanStatus = "created"
    PlanStatusExecuting  PlanStatus = "executing"
    PlanStatusCompleted  PlanStatus = "completed"
    PlanStatusFailed     PlanStatus = "failed"
    PlanStatusAborted    PlanStatus = "aborted"
)
```

#### 3.2 Execution Pattern Implementation

**Sequential Pattern:**
```go
func (e *AIConversationEngine) executeSequential(ctx context.Context, plan *ConversationPlan) (string, error) {
    var lastResponse *messaging.AgentToAIMessage
    
    for _, step := range plan.Steps {
        // Wait for dependencies to complete
        if err := e.waitForDependencies(ctx, plan, step); err != nil {
            return "", fmt.Errorf("dependency wait failed: %w", err)
        }
        
        // Execute step with context from previous steps
        response, err := e.executeStepWithContext(ctx, step, lastResponse)
        if err != nil {
            // AI decides: continue, retry, or abort
            decision := e.handleStepFailure(ctx, plan, step, err)
            if decision == "abort" {
                return "", fmt.Errorf("execution aborted: %w", err)
            }
        }
        
        lastResponse = response
        e.updateStepCompletion(ctx, step, response)
    }
    
    // AI processes all step results and generates final response
    return e.generateFinalResponse(ctx, plan)
}
```

**Parallel Pattern:**
```go  
func (e *AIConversationEngine) executeParallel(ctx context.Context, plan *ConversationPlan) (string, error) {
    parallelSteps := e.getParallelSteps(plan)
    responseChannels := make(map[string]chan *messaging.AgentToAIMessage)
    
    // Start all parallel steps
    for _, step := range parallelSteps {
        responseChannels[step.ID] = make(chan *messaging.AgentToAIMessage, 1)
        go e.executeStepAsync(ctx, step, responseChannels[step.ID])
    }
    
    // Wait for all parallel responses with timeout
    responses := e.waitForAllResponses(ctx, responseChannels, 5*time.Minute)
    
    // AI processes multiple responses and generates final answer
    return e.processMultipleResponses(ctx, plan, responses)
}
```

#### 3.3 Graph Storage for Plans

```go
// internal/graph/conversation_plan_store.go
type ConversationPlanStore struct {
    neo4jGraph *Neo4jGraph
}

func (s *ConversationPlanStore) StorePlan(ctx context.Context, plan *ConversationPlan) error {
    query := `
        CREATE (p:ConversationPlan {
            id: $id,
            user_id: $userId,
            user_request: $userRequest,
            ai_strategy: $aiStrategy,
            execution_pattern: $executionPattern,
            status: $status,
            created_at: $createdAt
        })
        WITH p
        UNWIND $steps AS step
        CREATE (s:ConversationStep {
            id: step.id,
            agent_id: step.agent_id,
            capability: step.capability,
            instruction: step.instruction,
            status: step.status
        })
        CREATE (p)-[:HAS_STEP]->(s)
    `
    
    return s.neo4jGraph.executeQuery(ctx, query, map[string]interface{}{
        "id": plan.ID,
        "userId": plan.UserID,
        "userRequest": plan.UserRequest,
        "aiStrategy": plan.AIStrategy,
        "executionPattern": string(plan.ExecutionPattern),
        "status": string(plan.Status),
        "createdAt": plan.CreatedAt,
        "steps": convertStepsToMap(plan.Steps),
    })
}
```

---

## 🗂️ PHASE 4: TECHNICAL DEBT CLEANUP

### Priority 6: Remove Over-Engineering

#### 4.1 Planning Domain Removal
```bash
# TDD Steps:
# 1. RED: Write test to ensure no planning domain usage
# 2. GREEN: Remove /internal/planning/ entirely
# 3. REFACTOR: Clean up any imports/references
# 4. VALIDATE: Ensure all tests still pass

rm -rf /mnt/c/Work/git/ztdp/orchestrator/internal/planning/
```

#### 4.2 LearningService Evaluation
```go
// Evaluate if LearningService provides value or is YAGNI
// Current usage analysis needed:
grep -r "LearningService" internal/
```

#### 4.3 Legacy Reference Cleanup
- Remove unused imports in main.go
- Clean up ServiceFactory legacy references  
- Remove complex ExecutionPlan domain if unused

---

## 📊 TESTING STRATEGY

### Per-Phase Testing Requirements

**Phase 2 Tests:**
- Agent heartbeat integration tests
- Registry health monitoring tests  
- Graph cleanup tests
- gRPC server update validation tests

**Phase 2.5 Tests:**
- UI streaming tests
- Browser automation tests
- End-to-end user journey tests

**Phase 3 Tests:**
- Multi-agent conversation tests
- Sequential execution pattern tests
- Parallel execution pattern tests
- Conditional execution pattern tests
- Plan storage and retrieval tests

**All Tests Must:**
- Use real AI provider (no MockAIProvider)
- Test with real event handling (no simulation)
- Follow TDD cycle: RED → GREEN → REFACTOR → VALIDATE
- Maintain clean architecture principles
- Apply SOLID principles throughout

---

## 🎯 SUCCESS CRITERIA

### Phase 2 Complete When:
- ✅ Agents send heartbeats every 30 seconds
- ✅ Registry marks disconnected agents correctly
- ✅ Graph cleanup removes stale agents
- ✅ gRPC server handles all new protobuf methods
- ✅ All tests pass with real AI and real events

### Phase 2.5 Complete When:
- ✅ Modern UI displays real-time conversations
- ✅ Agent status dashboard shows live updates
- ✅ Browser tests validate full user journey
- ✅ Streaming responses work correctly in UI

### Phase 3 Complete When:  
- ✅ Multi-agent conversations work end-to-end
- ✅ Sequential pattern executes build → test → deploy
- ✅ Parallel pattern handles independent tasks
- ✅ Conditional pattern adapts based on results
- ✅ Plans stored and retrieved from graph correctly

### Phase 4 Complete When:
- ✅ No over-engineered code remains
- ✅ YAGNI principles applied throughout
- ✅ Clean architecture maintained
- ✅ All technical debt resolved

---

## 🚀 READY FOR PRODUCTION

After all phases complete, the system will have:

1. **True AI-Native Orchestration**: Real AI making all decisions
2. **Event-Driven Architecture**: All communication via RabbitMQ events  
3. **Multi-Agent Coordination**: Complex workflows with multiple agents
4. **Health Monitoring**: Robust agent lifecycle management
5. **Modern UI**: Real-time conversation interface
6. **Clean Architecture**: SOLID principles, no over-engineering
7. **Full Test Coverage**: TDD throughout with real AI integration

**Estimated Timeline:**
- Phase 2: 1-2 weeks (infrastructure completion)
- Phase 2.5: 1 week (UI enhancement)  
- Phase 3: 2-3 weeks (multi-agent orchestration)
- Phase 4: 1 week (cleanup)

**Total: 5-7 weeks to full production readiness**
