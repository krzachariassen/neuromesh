# Comprehensive Graph Schema Implementation Plan

## 🎯 IMPLEMENTATION STRATEGY

This document outlines the TDD-driven implementation plan for the comprehensive graph schema based on the analysis in `COMPREHENSIVE_GRAPH_SCHEMA_ANALYSIS.md`.

## 📋 PHASE 1: CORE ENTITY SCHEMA EXTENSION (Immediate - 2-3 hours)

### **Goal**: Extend existing Agent/Capability schema with core user and conversation entities

### **TDD Implementation Steps**

#### **Step 1.1: User and Session Schema (RED-GREEN-REFACTOR)**

**RED**: Write failing tests for User and Session schema
```go
func TestGraphRepository_UserSchema(t *testing.T) {
    // Test user node creation with constraints
    // Test session node creation and user relationship
    // Test user-session relationship integrity
}
```

**GREEN**: Implement User and Session schema management
```go
// Add to graph_repository.go
func (gr *GraphRepository) EnsureUserSchema(ctx context.Context) error {
    queries := []string{
        // User node constraints and indexes
        "CREATE CONSTRAINT user_id_unique IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE",
        "CREATE INDEX user_session_idx IF NOT EXISTS FOR (u:User) ON (u.sessionId)",
        "CREATE INDEX user_type_idx IF NOT EXISTS FOR (u:User) ON (u.userType)",
        
        // Session node constraints and indexes
        "CREATE CONSTRAINT session_id_unique IF NOT EXISTS FOR (s:Session) REQUIRE s.id IS UNIQUE",
        "CREATE INDEX session_user_idx IF NOT EXISTS FOR (s:Session) ON (s.userId)",
        "CREATE INDEX session_status_idx IF NOT EXISTS FOR (s:Session) ON (s.status)",
    }
    
    return gr.executeSchemaQueries(ctx, queries)
}
```

**REFACTOR**: Clean up schema management interface

#### **Step 1.2: Conversation Schema (RED-GREEN-REFACTOR)**

**RED**: Write failing tests for Conversation schema
```go
func TestGraphRepository_ConversationSchema(t *testing.T) {
    // Test conversation node creation
    // Test conversation-user relationships
    // Test conversation message tracking
}
```

**GREEN**: Implement Conversation schema
```go
func (gr *GraphRepository) EnsureConversationSchema(ctx context.Context) error {
    queries := []string{
        "CREATE CONSTRAINT conversation_id_unique IF NOT EXISTS FOR (c:Conversation) REQUIRE c.id IS UNIQUE",
        "CREATE INDEX conversation_user_idx IF NOT EXISTS FOR (c:Conversation) ON (c.userId)",
        "CREATE INDEX conversation_status_idx IF NOT EXISTS FOR (c:Conversation) ON (c.status)",
        "CREATE INDEX conversation_updated_idx IF NOT EXISTS FOR (c:Conversation) ON (c.updatedAt)",
    }
    
    return gr.executeSchemaQueries(ctx, queries)
}
```

#### **Step 1.3: UserRequest and Analysis Schema (RED-GREEN-REFACTOR)**

**RED**: Write failing tests for UserRequest and Analysis
```go
func TestGraphRepository_RequestAnalysisSchema(t *testing.T) {
    // Test user request node creation
    // Test analysis node creation
    // Test request-analysis relationships
}
```

**GREEN**: Implement UserRequest and Analysis schema
```go
func (gr *GraphRepository) EnsureRequestAnalysisSchema(ctx context.Context) error {
    queries := []string{
        // UserRequest constraints and indexes
        "CREATE CONSTRAINT user_request_id_unique IF NOT EXISTS FOR (ur:UserRequest) REQUIRE ur.id IS UNIQUE",
        "CREATE INDEX user_request_user_idx IF NOT EXISTS FOR (ur:UserRequest) ON (ur.userId)",
        "CREATE INDEX user_request_session_idx IF NOT EXISTS FOR (ur:UserRequest) ON (ur.sessionId)",
        "CREATE INDEX user_request_status_idx IF NOT EXISTS FOR (ur:UserRequest) ON (ur.status)",
        
        // Analysis constraints and indexes
        "CREATE CONSTRAINT analysis_id_unique IF NOT EXISTS FOR (a:Analysis) REQUIRE a.id IS UNIQUE",
        "CREATE INDEX analysis_request_idx IF NOT EXISTS FOR (a:Analysis) ON (a.requestId)",
        "CREATE INDEX analysis_intent_idx IF NOT EXISTS FOR (a:Analysis) ON (a.intent)",
    }
    
    return gr.executeSchemaQueries(ctx, queries)
}
```

### **Files to Create/Modify**:
- `internal/conversation/infrastructure/graph_repository.go` - New repository for conversation entities
- `internal/conversation/infrastructure/graph_repository_test.go` - TDD tests
- `internal/graph/schema_manager.go` - Centralized schema management
- Update `internal/graph/neo4j_graph.go` - Extend schema initialization

## 📋 PHASE 2: AI DECISION & EXECUTION SCHEMA (Next - 2-3 hours)

### **Goal**: Add AI decision making and execution tracking to the graph

#### **Step 2.1: AIDecision Schema (RED-GREEN-REFACTOR)**

**RED**: Write failing tests for AIDecision schema
```go
func TestGraphRepository_AIDecisionSchema(t *testing.T) {
    // Test AI decision node creation
    // Test decision-request relationships
    // Test decision sequencing
}
```

**GREEN**: Implement AIDecision schema
```go
func (gr *GraphRepository) EnsureAIDecisionSchema(ctx context.Context) error {
    queries := []string{
        "CREATE CONSTRAINT ai_decision_id_unique IF NOT EXISTS FOR (ad:AIDecision) REQUIRE ad.id IS UNIQUE",
        "CREATE INDEX ai_decision_request_idx IF NOT EXISTS FOR (ad:AIDecision) ON (ad.requestId)",
        "CREATE INDEX ai_decision_status_idx IF NOT EXISTS FOR (ad:AIDecision) ON (ad.status)",
        "CREATE INDEX ai_decision_type_idx IF NOT EXISTS FOR (ad:AIDecision) ON (ad.type)",
        "CREATE INDEX ai_decision_created_idx IF NOT EXISTS FOR (ad:AIDecision) ON (ad.createdAt)",
    }
    
    return gr.executeSchemaQueries(ctx, queries)
}
```

#### **Step 2.2: ExecutionPlan and ExecutionStep Schema (RED-GREEN-REFACTOR)**

**RED**: Write failing tests for execution tracking
```go
func TestGraphRepository_ExecutionSchema(t *testing.T) {
    // Test execution plan node creation
    // Test execution step relationships
    // Test plan-agent assignments
}
```

**GREEN**: Implement execution schema
```go
func (gr *GraphRepository) EnsureExecutionSchema(ctx context.Context) error {
    queries := []string{
        // ExecutionPlan
        "CREATE CONSTRAINT execution_plan_id_unique IF NOT EXISTS FOR (ep:ExecutionPlan) REQUIRE ep.id IS UNIQUE",
        "CREATE INDEX execution_plan_user_idx IF NOT EXISTS FOR (ep:ExecutionPlan) ON (ep.userId)",
        "CREATE INDEX execution_plan_status_idx IF NOT EXISTS FOR (ep:ExecutionPlan) ON (ep.status)",
        
        // ExecutionStep
        "CREATE CONSTRAINT execution_step_id_unique IF NOT EXISTS FOR (es:ExecutionStep) REQUIRE es.id IS UNIQUE",
        "CREATE INDEX execution_step_plan_idx IF NOT EXISTS FOR (es:ExecutionStep) ON (es.planId)",
        "CREATE INDEX execution_step_agent_idx IF NOT EXISTS FOR (es:ExecutionStep) ON (es.agentId)",
        "CREATE INDEX execution_step_status_idx IF NOT EXISTS FOR (es:ExecutionStep) ON (es.status)",
    }
    
    return gr.executeSchemaQueries(ctx, queries)
}
```

### **Files to Create/Modify**:
- `internal/orchestrator/infrastructure/graph_decision_repository.go` - AI decision tracking
- `internal/planning/infrastructure/graph_execution_repository.go` - Execution tracking
- Tests for both repositories

## 📋 PHASE 3: MESSAGE & EVENT TRACKING SCHEMA (Following - 2-3 hours)

### **Goal**: Complete message flow and event tracking in the graph

#### **Step 3.1: Message Schema (RED-GREEN-REFACTOR)**

**RED**: Write failing tests for comprehensive message tracking
```go
func TestGraphRepository_MessageSchema(t *testing.T) {
    // Test all message types (AI-Agent, User-AI, etc.)
    // Test correlation tracking
    // Test message sequencing
}
```

**GREEN**: Implement comprehensive message schema
```go
func (gr *GraphRepository) EnsureMessageSchema(ctx context.Context) error {
    queries := []string{
        "CREATE CONSTRAINT message_id_unique IF NOT EXISTS FOR (m:Message) REQUIRE m.id IS UNIQUE",
        "CREATE INDEX message_correlation_idx IF NOT EXISTS FOR (m:Message) ON (m.correlationId)",
        "CREATE INDEX message_type_idx IF NOT EXISTS FOR (m:Message) ON (m.messageType)",
        "CREATE INDEX message_timestamp_idx IF NOT EXISTS FOR (m:Message) ON (m.timestamp)",
        "CREATE INDEX message_from_idx IF NOT EXISTS FOR (m:Message) ON (m.fromId)",
        "CREATE INDEX message_to_idx IF NOT EXISTS FOR (m:Message) ON (m.toId)",
    }
    
    return gr.executeSchemaQueries(ctx, queries)
}
```

#### **Step 3.2: Event and Correlation Schema (RED-GREEN-REFACTOR)**

**RED**: Write failing tests for event tracking and correlation
```go
func TestGraphRepository_EventCorrelationSchema(t *testing.T) {
    // Test event node creation and relationships
    // Test correlation context tracking
    // Test state transition tracking
}
```

**GREEN**: Implement event and correlation schema
```go
func (gr *GraphRepository) EnsureEventCorrelationSchema(ctx context.Context) error {
    queries := []string{
        // Event schema
        "CREATE CONSTRAINT event_id_unique IF NOT EXISTS FOR (e:Event) REQUIRE e.id IS UNIQUE",
        "CREATE INDEX event_type_idx IF NOT EXISTS FOR (e:Event) ON (e.type)",
        "CREATE INDEX event_entity_idx IF NOT EXISTS FOR (e:Event) ON (e.entityType, e.entityId)",
        "CREATE INDEX event_timestamp_idx IF NOT EXISTS FOR (e:Event) ON (e.timestamp)",
        
        // Correlation context
        "CREATE CONSTRAINT correlation_id_unique IF NOT EXISTS FOR (cc:CorrelationContext) REQUIRE cc.id IS UNIQUE",
        "CREATE INDEX correlation_user_idx IF NOT EXISTS FOR (cc:CorrelationContext) ON (cc.userId)",
        "CREATE INDEX correlation_status_idx IF NOT EXISTS FOR (cc:CorrelationContext) ON (cc.status)",
    }
    
    return gr.executeSchemaQueries(ctx, queries)
}
```

### **Files to Create/Modify**:
- `internal/messaging/infrastructure/graph_message_repository.go` - Message tracking
- `internal/events/infrastructure/graph_event_repository.go` - Event tracking (new module)
- Update existing message bus to store messages in graph

## 📋 PHASE 4: COMPLETE GRAPH INTEGRATION (Final - 3-4 hours)

### **Goal**: Integrate all components to use the graph as primary state store

#### **Step 4.1: Repository Pattern Implementation (RED-GREEN-REFACTOR)**

**RED**: Write failing tests for complete graph integration
```go
func TestGraphIntegration_EndToEnd(t *testing.T) {
    // Test complete user request flow through graph
    // Test AI decision making with graph context
    // Test agent execution tracking
}
```

**GREEN**: Implement complete graph integration
- Update all application services to use graph repositories
- Ensure all state changes are persisted to graph
- Implement graph-based context retrieval

#### **Step 4.2: Graph-Native AI Context (RED-GREEN-REFACTOR)**

**RED**: Write failing tests for AI context from graph
```go
func TestAIContextProvider_GraphBased(t *testing.T) {
    // Test user context retrieval from graph
    // Test conversation history from graph
    // Test agent performance data from graph
}
```

**GREEN**: Implement graph-native AI context provider
```go
type GraphAIContextProvider struct {
    graph graph.Graph
}

func (g *GraphAIContextProvider) GetUserContext(ctx context.Context, userID string) (*UserContext, error) {
    // Query graph for complete user context
    // Include conversation history, preferences, patterns
}

func (g *GraphAIContextProvider) GetAgentContext(ctx context.Context) (string, error) {
    // Query graph for real-time agent status and capabilities
    // Include performance metrics and availability
}
```

### **Files to Create/Modify**:
- `internal/graph/context_provider.go` - Graph-native context provider
- Update all application services to use graph repositories
- Integration tests for complete flows

## 🔧 **IMPLEMENTATION GUIDELINES**

### **TDD Enforcement Protocol**
1. **RED**: Write failing test that exposes what's missing
2. **GREEN**: Write minimal code to make test pass
3. **REFACTOR**: Clean up while keeping tests green
4. **VALIDATE**: Run all tests to ensure nothing breaks
5. **REPEAT**: Never skip the cycle

### **Clean Architecture Compliance**
- **Domain**: Keep entity definitions pure (no infrastructure concerns)
- **Application**: Use case implementations with interface dependencies
- **Infrastructure**: Graph repository implementations
- **Interfaces**: Clear boundaries between layers

### **Graph Design Principles**
- **Nodes**: Represent entities with properties
- **Relationships**: Represent meaningful connections
- **Indexes**: Optimize for query patterns
- **Constraints**: Ensure data integrity
- **Properties**: Store relevant state information

### **Performance Considerations**
- Use indexes for high-frequency queries
- Implement batching for bulk operations
- Cache frequently accessed data
- Monitor query performance
- Optimize relationship directions

## 📊 **SUCCESS CRITERIA**

### **Phase 1 Complete**
- ✅ All core entities (User, Session, Conversation, UserRequest, Analysis) in graph
- ✅ Basic relationships established
- ✅ Schema constraints and indexes in place
- ✅ All tests passing

### **Phase 2 Complete**
- ✅ AI decision tracking fully implemented
- ✅ Execution plan and step tracking
- ✅ Agent assignment and status tracking
- ✅ Decision flow relationships

### **Phase 3 Complete**
- ✅ Complete message flow tracking in graph
- ✅ Event-driven state changes captured
- ✅ Correlation context for async operations
- ✅ Audit trail for all operations

### **Phase 4 Complete**
- ✅ All application services use graph as primary store
- ✅ AI gets complete context from graph
- ✅ Real-time state reflected in graph
- ✅ Complete memory and audit trail
- ✅ Performance optimized for production

## 🎯 **NEXT ACTIONS**

1. **Start with Phase 1, Step 1.1**: Implement User and Session schema
2. **Follow TDD religiously**: RED-GREEN-REFACTOR for every change
3. **Validate incrementally**: Ensure each step works before proceeding
4. **Document progress**: Update status as each phase completes
5. **Performance test**: Ensure graph queries are optimized

This implementation plan ensures a systematic, TDD-driven approach to creating a comprehensive graph schema that captures every aspect of the AI system's operation and decision-making process.
