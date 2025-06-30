# Correlation-Based Async Architecture Refactor

**Date:** June 30, 2025  
**Status:** Planning Phase  
**Priority:** Critical (Scalability Blocker)

## 🎯 **Objective**
Refactor the AI orchestration system from a blocking, single-threaded design to a fully async, correlation-based architecture that supports:
- Multiple concurrent users
- Parallel multi-agent execution  
- True scalability without RabbitMQ queue proliferation
- Stateless service design

## 🚨 **Current Problems**
1. **Single Shared Channel**: All users compete for same `responseChannel` in `AIConversationEngine`
2. **Blocking Design**: `waitForAgentResponse()` blocks threads, preventing concurrency
3. **Instance State**: `conversationID` stored as instance variable, causes race conditions
4. **No Multi-Agent Support**: Cannot coordinate multiple agents in parallel
5. **Poor Timeout Handling**: Messages can be lost or mis-routed between users

## 🏗️ **Target Architecture**
```
User Request → Generate CorrelationID → Flow Through System → Route Response Back
     ↓                    ↓                     ↓                      ↓
WebBFF API        AI Orchestrator         Agent Processing    Correlation Router
(Entry Point)    (Business Logic)        (Async Execution)    (Response Routing)
```

## 📋 **Implementation Backlog**

### **Phase 1: Foundation Components**
- [ ] **1.1** Create `CorrelationTracker` service
  - `RegisterRequest(correlationID, userID, timeout)` 
  - `RouteResponse(response)` 
  - `CleanupRequest(correlationID)`
  - `StartCleanupWorker()` for auto-cleanup
  - Location: `internal/orchestrator/infrastructure/correlation_tracker.go`

- [ ] **1.2** Create `GlobalMessageConsumer` service  
  - Single consumer for "ai-orchestrator" queue
  - Routes messages via correlation ID to waiting requests
  - Handles unknown correlation IDs gracefully
  - Location: `internal/orchestrator/infrastructure/global_consumer.go`

- [ ] **1.3** Update messaging interfaces
  - Ensure correlation ID flows through all message types
  - Add logging for correlation ID tracking
  - Verify message parsing maintains correlation context

### **Phase 2: API Layer Changes**
- [ ] **2.1** Update WebBFF to generate correlation IDs
  - Generate at API boundary: `web-{sessionID}-{timestamp}`
  - Pass correlation ID through orchestrator calls
  - Add correlation ID to response for client tracking
  - File: `internal/web/bff.go`

- [ ] **2.2** Update OrchestratorService interface
  - Add `CorrelationID` field to `OrchestratorRequest`
  - Pass correlation ID to conversation engine
  - Update all method signatures
  - File: `internal/orchestrator/application/orchestrator_service.go`

- [ ] **2.3** Update orchestrator result structures
  - Include correlation ID in responses
  - Ensure clean error handling with correlation context

### **Phase 3: Core Engine Refactor**
- [ ] **3.1** Refactor AIConversationEngine to be stateless
  - **REMOVE**: `conversationID`, `responseChannel`, `subscriptionOnce`, `channelMutex`
  - **ADD**: `tracker *CorrelationTracker` dependency
  - **MODIFY**: All methods to accept correlation ID as parameter
  - File: `internal/orchestrator/application/ai_conversation_engine.go`

- [ ] **3.2** Update ProcessWithAgents method
  - Accept correlation ID as parameter (don't generate internally)
  - Pass correlation ID to all subsequent calls
  - Remove blocking wait logic

- [ ] **3.3** Refactor handleAgentEvent method
  - Use existing correlation ID instead of instance variable
  - Register with tracker BEFORE sending to agent
  - Use tracker's response channel instead of direct waiting

- [ ] **3.4** Remove blocking response methods
  - **DELETE**: `ensureSubscription()`, `waitForAgentResponse()`
  - Replace with tracker-based async response handling

### **Phase 4: Dependency Injection & Wiring**
- [ ] **4.1** Update ServiceFactory
  - Create and inject `CorrelationTracker`
  - Create and start `GlobalMessageConsumer` 
  - Wire tracker into `AIConversationEngine`
  - Ensure proper startup order
  - File: `internal/orchestrator/application/service_factory.go`

- [ ] **4.2** Update all constructors
  - Add tracker dependencies where needed
  - Ensure clean dependency injection
  - Update tests to use mock tracker

- [ ] **4.3** Add graceful shutdown
  - Stop global consumer cleanly
  - Cleanup pending requests on shutdown
  - Ensure no message loss during shutdown

### **Phase 5: Testing & Validation**
- [ ] **5.1** Unit Tests
  - Test `CorrelationTracker` request/response matching
  - Test `GlobalMessageConsumer` routing logic
  - Test concurrent request handling
  - Test timeout and cleanup behavior

- [ ] **5.2** Integration Tests  
  - Multiple concurrent users scenario
  - Agent timeout handling
  - Message loss prevention
  - Correlation ID flow validation

- [ ] **5.3** Load Testing
  - 10+ concurrent users
  - Multiple agents per request
  - Sustained load over time
  - Memory leak detection

### **Phase 6: Multi-Agent Preparation**
- [ ] **6.1** Extend CorrelationTracker for multi-agent
  - Track multiple agents per correlation ID
  - Collect responses from multiple agents
  - Handle partial agent failures
  - Coordinate agent dependency chains

- [ ] **6.2** Update AI prompting for multi-agent
  - Support parallel agent execution
  - Handle agent result aggregation
  - Implement agent dependency resolution

- [ ] **6.3** Error handling for multi-agent scenarios
  - Partial success handling
  - Agent failure compensation
  - Timeout handling for agent groups

## 🎯 **Success Criteria**
- [ ] System handles 10+ concurrent users without message mixing
- [ ] No blocking threads during agent communication
- [ ] All correlation IDs flow correctly through system
- [ ] Automatic cleanup of expired requests
- [ ] Single RabbitMQ queue serves all users efficiently
- [ ] Memory usage remains constant under load
- [ ] Foundation ready for multi-agent coordination

## 🚨 **Risks & Mitigation**
1. **Message Loss**: Implement robust error handling and cleanup
2. **Memory Leaks**: Ensure all pending requests are cleaned up
3. **Deadlocks**: Remove all blocking operations
4. **Race Conditions**: Use proper synchronization in tracker
5. **Performance**: Benchmark against current implementation

## 📊 **Testing Strategy**
1. **Unit**: Test each component in isolation
2. **Integration**: Test full flow with real agents
3. **Concurrent**: Multiple users simultaneously
4. **Stress**: High message volume and sustained load
5. **Chaos**: Random failures and network issues

## 📝 **Implementation Notes**
- Keep current system working during refactor
- Implement behind feature flag if possible
- Maintain backward compatibility during transition
- Document all correlation ID flows
- Add extensive logging for debugging

## 🔄 **Rollback Plan**
- Git branch: `feature/correlation-async-refactor`
- Current working state committed before refactor starts
- Ability to revert to blocking implementation if needed
- Keep current code as backup until full validation

---

**Next Steps:**
1. Commit current working state
2. Create feature branch
3. Start with Phase 1.1 - CorrelationTracker implementation
4. Test each phase before proceeding to next
