# 🎯 NEXT STEP: Planning Domain Fix & Orchestrator Graph Persistence

**Status**: Ready for Implementation  
**Priority**: P0 - Immediate Fix Required  
**Last Updated**: July 4, 2025

## 🚧 IMMEDIATE BLOCKER: Planning Domain Compilation

**Issue**: Planning domain has parameter mismatches preventing compilation
- `domain.NewAnalysis()` expects `requestID` parameter 
- `domain.NewClarifyDecision()` expects `requestID` and `analysisID` parameters
- Planning domain currently generates requestID instead of using messageID from conversation

**Impact**: Blocking all orchestrator graph persistence work

## ✅ SOLUTION IDENTIFIED: Thread MessageID Through Orchestrator

The `requestID` should be the **messageID** from the conversation system to properly link orchestrator decisions to specific messages in conversations.

**Flow**: ConversationBFF creates message → passes messageID → orchestrator uses as requestID → planning domain links to conversation

## 🔧 EXACT IMPLEMENTATION STEPS

### Step 1: Add MessageID to OrchestratorRequest  
File: `/internal/orchestrator/application/orchestrator_service.go`
```go
type OrchestratorRequest struct {
    UserInput   string `json:"user_input"`
    UserID      string `json:"user_id"`
    SessionID   string `json:"session_id,omitempty"`
    MessageID   string `json:"message_id"`  // ADD THIS
}
```

### Step 2: Update Interface Signatures
Update `AIDecisionEngineInterface` to accept `requestID` parameter:
```go
type AIDecisionEngineInterface interface {
    ExploreAndAnalyze(ctx context.Context, userInput, userID, agentContext, requestID string) (*orchestratorDomain.Analysis, error)
    MakeDecision(ctx context.Context, userInput, userID string, analysis *orchestratorDomain.Analysis, requestID string) (*orchestratorDomain.Decision, error)
}
```

### Step 3: Update Planning Domain Implementation
File: `/internal/planning/application/ai_decision_engine.go`
- Add `requestID` parameter to method signatures
- Remove auto-generated requestID line  
- Use passed requestID in domain constructors

### Step 4: Update ConversationBFF
File: `/internal/web/conversation_bff.go`
- Pass `userMessageID` as `MessageID` in OrchestratorRequest

### Step 5: Update Orchestrator Service Calls
- Pass `request.MessageID` to planning domain methods

## 🚀 POST-FIX IMPLEMENTATION ROADMAP

### Phase 1: Analysis Domain Graph Persistence (TDD)
**Objective**: Store AI analysis results in graph with proper relationships

**Implementation**:
1. **RED**: Write failing tests for Analysis graph repository
2. **GREEN**: Implement minimal Analysis repository with Neo4j backend  
3. **REFACTOR**: Clean up Analysis domain persistence integration
4. **VALIDATE**: Ensure all tests pass and analysis data persists correctly

**Files to Create**:
- `/internal/planning/domain/analysis_repository.go` - Repository interface
- `/internal/planning/infrastructure/graph_analysis_repository.go` - Neo4j implementation  
- `/internal/planning/infrastructure/graph_analysis_repository_test.go` - TDD tests

### Phase 2: Decision Domain Graph Persistence (TDD)
**Objective**: Store AI decisions in graph linked to analysis and conversation

**Similar pattern to Analysis domain** with Decision-specific repository

### Phase 3: Execution Domain Graph Persistence (TDD)  
**Objective**: Store execution plans and steps with agent relationships

**Most complex domain** - ExecutionPlan and ExecutionStep with agent coordination tracking

### Phase 4: End-to-End Integration Testing
**Objective**: Validate complete orchestrator flow with graph persistence

## 📚 REFERENCE DOCUMENTS

**Essential Reading**:
1. `/docs/ORCHESTRATOR_GRAPH_PERSISTENCE_ANALYSIS.md` - Complete technical analysis
2. `/docs/IMPLEMENTATION_BACKLOG.md` - Detailed implementation roadmap  
3. `/docs/VACATION_RESUME_POINT.md` - Complete context for resuming work

**Working Examples**:
- `/internal/conversation/` - Complete graph persistence implementation (follow this pattern)
- `/internal/user/` - Clean repository architecture example

## 🎯 SUCCESS METRICS

### Immediate Success (Planning Domain Fix)
- [ ] Go compilation succeeds without errors
- [ ] Planning domain tests pass
- [ ] MessageID flows correctly through orchestrator
- [ ] Analysis/Decision objects created with proper requestID linking to messages

### Complete Success (All Phases)  
- [ ] Full orchestrator intelligence persisted in graph
- [ ] AI decision traceability from conversation to execution
- [ ] Rich data available for learning and optimization
- [ ] Clean architecture maintained with TDD throughout

**Next Action**: Apply the 5-step fix above, then proceed with systematic TDD implementation of graph persistence for each orchestrator domain.
