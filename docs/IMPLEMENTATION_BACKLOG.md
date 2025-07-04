# Implementation Backlog - NeuroMesh Graph Persistence

## Current Phase: Orchestrator Graph Persistence Implementation

### COMPLETED ✅
1. **User Domain Implementation** - Full graph persistence with clean architecture
2. **Session Domain Implementation** - Complete with Neo4j backend  
3. **Conversation Domain Cleanup** - Removed legacy code, implemented clean graph persistence
4. **ConversationAwareWebBFF Integration** - Production-ready conversation tracking
5. **Neo4j Property Type Fixes** - Resolved metadata and array persistence issues
6. **Graph Schema Documentation** - Complete conversation schema documented
7. **End-to-End Testing** - All conversation flows validated with tests
8. **Orchestrator Flow Analysis** - Complete end-to-end analysis of ProcessUserRequest flow
9. **Graph Schema Design** - Analysis, Decision, ExecutionPlan schemas defined

### IN PROGRESS 🚧
1. **Planning Domain Compilation Fixes** - Fixing requestID parameter mismatches
   - Issue: `domain.NewAnalysis()` and `domain.NewClarifyDecision()` expect different parameters
   - Solution: Pass messageID from conversation as requestID through orchestrator flow
   - Status: Identified the fix, need to implement parameter threading

### NEXT PRIORITY 🎯
1. **Fix Planning Domain Compilation Issues** - Complete the requestID fix
2. **Planning Domain Graph Persistence** - Implement Analysis repository with TDD
3. **Decision Domain Graph Persistence** - Implement Decision repository with TDD  
4. **Execution Domain Graph Persistence** - Implement ExecutionPlan repository with TDD
5. **End-to-End Integration Testing** - Full orchestrator flow with graph persistence

## Detailed Implementation Plan

### Phase 1: Fix Planning Domain (IMMEDIATE)
**Files to modify:**
- `/internal/orchestrator/application/orchestrator_service.go` - Add MessageID parameter
- `/internal/planning/application/ai_decision_engine.go` - Accept requestID parameter
- `/internal/web/conversation_bff.go` - Pass messageID to orchestrator
- Update interface signatures for consistency

**Steps:**
1. Add `MessageID` field to `OrchestratorRequest`
2. Update `ExploreAndAnalyze` and `MakeDecision` signatures to accept `requestID`
3. Thread messageID from ConversationAwareWebBFF through orchestrator to planning
4. Fix domain constructor calls with proper parameters
5. Run tests to validate compilation

### Phase 2: Analysis Domain Graph Persistence (RED/GREEN/REFACTOR)
**TDD Implementation:**
1. **RED:** Write failing tests for Analysis graph repository
2. **GREEN:** Implement minimal Analysis repository
3. **REFACTOR:** Clean up Analysis domain persistence
4. **VALIDATE:** Ensure all tests pass

**Files to create/modify:**
- `/internal/planning/domain/analysis_repository.go` - Repository interface
- `/internal/planning/infrastructure/graph_analysis_repository.go` - Neo4j implementation
- `/internal/planning/infrastructure/graph_analysis_repository_test.go` - TDD tests
- Update planning application to use repository

### Phase 3: Decision Domain Graph Persistence (RED/GREEN/REFACTOR)
**Similar pattern to Analysis domain:**
- Create Decision repository interface and Neo4j implementation
- Link Decision nodes to Analysis nodes in graph
- Full TDD implementation with tests

### Phase 4: Execution Domain Graph Persistence (RED/GREEN/REFACTOR)
**Most complex domain:**
- ExecutionPlan and ExecutionStep repositories
- Complex relationships to Decision, Agent nodes
- Status tracking and timing persistence
- Agent coordination tracking

### Phase 5: End-to-End Integration
- Full orchestrator flow testing
- Performance optimization
- Documentation updates

## Technical Debt & Cleanup
1. **Remove Learning Service References** - Following YAGNI principles
2. **Standardize Error Handling** - Consistent across all domains
3. **Optimize Graph Queries** - Performance improvements
4. **Documentation Updates** - Keep all docs current

## Architecture Principles
- **TDD Enforcement** - RED/GREEN/REFACTOR for all changes
- **SOLID Principles** - Clean architecture throughout
- **YAGNI Compliance** - Only implement what's needed now
- **Clean Architecture** - Domain boundaries strictly maintained
- **Graph-Native Design** - Everything persisted with proper relationships

## Current Blockers
1. **Planning Domain Compilation** - requestID parameter mismatch (immediate fix needed)

## Success Metrics
- All orchestrator data persisted in graph with proper relationships
- Full traceability from User → Session → Conversation → Message → Analysis → Decision → Execution
- Clean, testable code following architectural principles
- Performance benchmarks met for graph operations
