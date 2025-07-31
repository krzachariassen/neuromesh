# 🎯 **USER-AI INTERACTION WITH RICH CONTEXT - DEVELOPMENT PLAN**

## **📋 FEATURE OVERVIEW**

**Goal**: Enable intelligent user-AI interaction where the AI can ask clarification questions when encountering capability gaps or ambiguous requests, with full conversation context preserved in the graph.

**Core Problem**: Currently, when the AI planning engine encounters missing capabilities (e.g., no translation agent available), it creates workaround execution steps instead of asking the user for clarification and offering alternatives.

**Target Behavior**: AI should detect capability gaps, ask user for clarification with context, and store the full reasoning chain in the graph for rich conversation history.

**Real Example from Production**:
```
User: "Count words and translate to Spanish: 'the quick brown fox'"
Current AI Behavior: Creates workaround step "Analyze text properties to assist in further processing (note: translation not supported by available agents)"
Desired AI Behavior: "I can count the words for you (4 words), but no translation agent is available. Would you like to proceed with just word counting?"
```

---

## **🏗️ ARCHITECTURE ANALYSIS**

### **Current State Assessment**:
- ✅ **Foundation**: CLARIFY vs EXECUTE planning types exist in `PlanningType`
- ✅ **Infrastructure**: ExecutionPlan with AI reasoning storage
- ✅ **Conversation System**: ConversationMessage storage in graph
- ✅ **Orchestrator Flow**: Handles CLARIFY type in `orchestrator_service.go:123`
- ❌ **Gap**: AI planning doesn't effectively detect capability gaps
- ❌ **Gap**: No structured user clarification flow
- ❌ **Gap**: AI reasoning not fully stored in execution plans
- ❌ **Gap**: No conversation continuation after clarification

### **Key Files & Components**:
1. **AI Planning Engine**: `internal/planning/application/ai_planning_engine.go`
   - Current: Basic CLARIFY/EXECUTE logic
   - Needs: Enhanced capability gap detection

2. **Orchestrator Service**: `internal/orchestrator/application/orchestrator_service.go`
   - Current: Handles CLARIFY in `result.Message = planningResult.Reasoning`
   - Needs: Rich clarification message formatting

3. **ExecutionPlan Domain**: `internal/planning/domain/execution_plan.go`
   - Current: Has `Reasoning` field for AI analysis
   - Needs: Enhanced reasoning structure with capability gaps

4. **Conversation Domain**: `internal/conversation/domain/conversation.go`
   - Current: Basic message storage
   - Needs: Rich context preservation for clarification chains

### **Technical Dependencies**:
1. **Planning Domain**: ExecutionPlan, PlanningType, AI reasoning storage
2. **Conversation Domain**: ConversationMessage, rich context preservation  
3. **Orchestrator**: Request/response flow coordination
4. **AI Planning Engine**: Enhanced capability gap detection
5. **Graph Storage**: Cross-domain relationship management

---

## **📋 STRUCTURED DEVELOPMENT BACKLOG**

### **🔴 PHASE 1: Enhanced AI Planning Intelligence (TDD - 2-3 hours)**
*Status: PENDING*

#### **Epic 1.1: Intelligent Capability Gap Detection**
- **User Story**: As an AI planning engine, I need to detect when required capabilities are missing from available agents so I can ask users for clarification instead of creating workaround steps.
- **Acceptance Criteria**:
  - AI analyzes user request against available agent capabilities
  - Detects specific capability gaps (e.g., "translation" capability missing)
  - Chooses CLARIFY when gaps exist, not workaround EXECUTE plans
  - Stores detailed reasoning about capability gaps

**Tasks**:
- [ ] **Task 1.1.1**: Write failing test for capability gap detection
  - *File*: `internal/planning/application/ai_planning_engine_test.go`
  - *Test*: AI should return CLARIFY when translation requested but no translation agent available
  - *Expected Result*: `PlanningTypeClarify` with specific gap reasoning

- [ ] **Task 1.1.2**: Enhance AI prompt to include capability gap analysis
  - *File*: `internal/planning/application/ai_planning_engine.go`
  - *Target*: Improve system prompt to explicitly check for capability gaps
  - *New Logic*: Add capability gap detection section to AI prompt

- [ ] **Task 1.1.3**: Update response parser to extract capability gaps
  - *File*: `internal/planning/domain/response_parser.go`
  - *Target*: Parse capability gap information from AI responses
  - *New Fields*: Extract missing capabilities and suggested alternatives

- [ ] **Task 1.1.4**: Implement gap detection logic in planning engine
  - *File*: `internal/planning/application/ai_planning_engine.go`
  - *Target*: `CreateExecutionPlan` method enhancement
  - *Logic*: Compare required capabilities vs available agent capabilities

- [ ] **Task 1.1.5**: Test with real scenarios (word count + translation)
  - *Test Scenario*: "Count words and translate to Spanish: 'text'"
  - *Agent Setup*: Only text-processor agent available (no translator)
  - *Expected*: CLARIFY response with specific capability gap information

#### **Epic 1.2: Enhanced Planning Reasoning Storage**
- **User Story**: As a system administrator, I need full AI reasoning stored in ExecutionPlan so I can understand why specific planning decisions were made.
- **Acceptance Criteria**:
  - Complete AI reasoning chain stored in ExecutionPlan.Reasoning
  - Capability gap analysis included in reasoning
  - Decision rationale clearly documented
  - Reasoning accessible for conversation context

**Tasks**:
- [ ] **Task 1.2.1**: Extend ExecutionPlan.Reasoning field documentation
  - *File*: `internal/planning/domain/execution_plan.go`
  - *Target*: Add structured reasoning format documentation
  - *Fields*: Capability analysis, gap detection, decision rationale

- [ ] **Task 1.2.2**: Write test for comprehensive reasoning storage
  - *File*: `internal/planning/domain/execution_plan_test.go`
  - *Test*: Reasoning should include capability gaps and alternatives
  - *Validation*: Check reasoning completeness and structure

- [ ] **Task 1.2.3**: Update AI planning engine to store detailed reasoning
  - *File*: `internal/planning/application/ai_planning_engine.go`
  - *Target*: Enhanced reasoning extraction and storage
  - *Format*: Structured reasoning with capability analysis

- [ ] **Task 1.2.4**: Validate reasoning persistence in graph
  - *Test*: Ensure reasoning is properly stored and retrievable
  - *Integration*: Test graph storage and retrieval of enhanced reasoning

---

### **🟡 PHASE 2: Structured User Clarification Flow (TDD - 2-3 hours)**
*Status: PENDING*

#### **Epic 2.1: Rich Clarification Question Generation**
- **User Story**: As a user, I need to receive clear, contextual clarification questions when the AI encounters capability gaps, with specific options provided.
- **Acceptance Criteria**:
  - AI generates specific clarification questions (not generic responses)
  - Questions include context about missing capabilities
  - Alternative options presented to user
  - Questions stored as ConversationMessage

**Tasks**:
- [ ] **Task 2.1.1**: Design clarification question format
  - *Format*: Structured clarification with context and options
  - *Example*: "I can count words (4 words detected), but translation requires a translation agent. Options: 1) Proceed with word count only, 2) Wait for translation agent"

- [ ] **Task 2.1.2**: Write test for clarification message generation
  - *File*: `internal/orchestrator/application/orchestrator_service_test.go`
  - *Test*: CLARIFY type should generate rich clarification message
  - *Validation*: Message includes context, options, and clear next steps

- [ ] **Task 2.1.3**: Implement structured clarification in planning engine
  - *File*: `internal/planning/application/ai_planning_engine.go`
  - *Enhancement*: Generate structured clarification with alternatives
  - *AI Prompt*: Include clarification formatting instructions

- [ ] **Task 2.1.4**: Update orchestrator to handle clarification messages
  - *File*: `internal/orchestrator/application/orchestrator_service.go`
  - *Target*: Line 125 - Enhance clarification message formatting
  - *Current*: `result.Message = planningResult.Reasoning`
  - *New*: Rich clarification message with options

- [ ] **Task 2.1.5**: Test clarification question quality
  - *Validation*: Manual testing of clarification question clarity
  - *Criteria*: Questions are specific, actionable, and user-friendly

#### **Epic 2.2: Conversation Context Preservation**
- **User Story**: As a conversation system, I need to preserve full context of clarification exchanges so users can see the complete interaction history.
- **Acceptance Criteria**:
  - Original user request stored as ConversationMessage
  - AI clarification question stored as ConversationMessage
  - Planning reasoning linked to conversation
  - Full context retrievable for follow-up interactions

**Tasks**:
- [ ] **Task 2.2.1**: Design conversation context schema
  - *Schema*: ConversationMessage with clarification metadata
  - *Fields*: Message type, context references, planning decision links

- [ ] **Task 2.2.2**: Write test for context preservation
  - *Test*: Complete clarification exchange should be stored in graph
  - *Validation*: All messages linked with proper relationships

- [ ] **Task 2.2.3**: Implement context storage in orchestrator
  - *File*: `internal/orchestrator/application/orchestrator_service.go`
  - *Enhancement*: Store clarification context in conversation
  - *Integration*: Link planning decisions to conversation messages

- [ ] **Task 2.2.4**: Test context retrieval and display
  - *Test*: Retrieve full clarification context from graph
  - *UI Integration*: Test conversation history display

---

### **🟢 PHASE 3: User Response Processing (TDD - 2-3 hours)**
*Status: PENDING*

#### **Epic 3.1: Clarification Response Handling**
- **User Story**: As a user, when I respond to an AI clarification question, the system should understand my response and proceed accordingly.
- **Acceptance Criteria**:
  - User responses to clarifications properly parsed
  - AI understands user decisions (proceed with available, wait for agent, etc.)
  - New execution plan generated based on user response
  - Conversation continues seamlessly

**Tasks**:
- [ ] **Task 3.1.1**: Design user response parsing logic
  - *Logic*: Parse user responses to clarification questions
  - *Patterns*: "proceed with word count", "wait for translator", "option 1"

- [ ] **Task 3.1.2**: Write test for response understanding
  - *Test*: AI should understand various user response formats
  - *Scenarios*: Different ways users might respond to clarifications

- [ ] **Task 3.1.3**: Implement response processing in planning engine
  - *File*: `internal/planning/application/ai_planning_engine.go`
  - *New Method*: ProcessClarificationResponse
  - *Logic*: Generate execution plan based on user choice

- [ ] **Task 3.1.4**: Update orchestrator flow for follow-up planning
  - *File*: `internal/orchestrator/application/orchestrator_service.go`
  - *Enhancement*: Handle clarification responses
  - *Flow*: Detect response to clarification and process accordingly

- [ ] **Task 3.1.5**: Test complete clarification cycle
  - *Integration Test*: End-to-end clarification and response cycle
  - *Validation*: User request → clarification → response → execution

#### **Epic 3.2: Adaptive Plan Generation**
- **User Story**: As an AI planning engine, I need to generate appropriate execution plans based on user clarification responses.
- **Acceptance Criteria**:
  - Generate modified plans based on user decisions
  - Handle "proceed with available capabilities" scenarios
  - Handle "wait for missing capabilities" scenarios
  - Store relationship between original request and final plan

**Tasks**:
- [ ] **Task 3.2.1**: Design adaptive planning logic
  - *Logic*: Generate plans based on user clarification responses
  - *Scenarios*: Available capabilities only, wait for agents, alternatives

- [ ] **Task 3.2.2**: Write test for adaptive plan generation
  - *Test*: Different user responses generate appropriate plans
  - *Validation*: Plans match user decisions and available capabilities

- [ ] **Task 3.2.3**: Implement plan modification capabilities
  - *File*: `internal/planning/application/ai_planning_engine.go`
  - *Method*: GenerateAdaptivePlan
  - *Input*: Original request + user clarification response

- [ ] **Task 3.2.4**: Test various user response scenarios
  - *Scenarios*: Multiple clarification response types
  - *Validation*: Appropriate execution plans generated

---

### **🔵 PHASE 4: Graph Integration & Rich Context (TDD - 1-2 hours)**
*Status: PENDING*

#### **Epic 4.1: Cross-Domain Relationship Enhancement**
- **User Story**: As a graph database, I need to properly link clarification exchanges, planning decisions, and execution outcomes for complete traceability.
- **Acceptance Criteria**:
  - Clarification messages linked to original requests
  - Planning reasoning linked to conversations
  - Execution plans linked to clarification outcomes
  - Full interaction chain queryable

**Tasks**:
- [ ] **Task 4.1.1**: Design graph relationship schema
  - *Schema*: Clarification relationships in graph
  - *Relationships*: Request→Clarification→Response→Plan→Execution

- [ ] **Task 4.1.2**: Write test for relationship creation
  - *Test*: Complete clarification chain stored in graph
  - *Validation*: All relationships properly created and queryable

- [ ] **Task 4.1.3**: Implement relationship storage
  - *Files*: Graph repository implementations
  - *Enhancement*: Store clarification relationships

- [ ] **Task 4.1.4**: Test relationship queries
  - *Test*: Query complete clarification chains from graph
  - *Performance*: Ensure efficient relationship traversal

#### **Epic 4.2: Conversation History Enrichment**
- **User Story**: As a user interface, I need to display rich conversation history including AI reasoning and planning decisions.
- **Acceptance Criteria**:
  - Complete conversation history accessible
  - AI reasoning displayed when relevant
  - Planning decisions visible in UI
  - Context preserved across sessions

**Tasks**:
- [ ] **Task 4.2.1**: Design conversation history API
  - *API*: Enriched conversation history retrieval
  - *Data*: Messages + planning decisions + reasoning

- [ ] **Task 4.2.2**: Write test for history retrieval
  - *Test*: Complete conversation context retrievable
  - *Validation*: All clarification context included

- [ ] **Task 4.2.3**: Implement history enrichment
  - *Enhancement*: Add planning context to conversation history
  - *Integration*: Link planning decisions to conversation display

- [ ] **Task 4.2.4**: Test UI integration
  - *UI Test*: Conversation history displays clarification context
  - *UX*: Rich context improves user experience

---

### **🟣 PHASE 5: End-to-End Validation (TDD - 1-2 hours)**
*Status: PENDING*

#### **Epic 5.1: Real-World Scenario Testing**
- **User Story**: As a system, I need to handle real-world clarification scenarios end-to-end.
- **Acceptance Criteria**:
  - Word count + translation scenario works correctly
  - AI asks for clarification when translation agent missing
  - User can choose to proceed with word count only
  - Full interaction stored in graph

**Tasks**:
- [ ] **Task 5.1.1**: Implement real-world test scenario
  - *Scenario*: "Count words and translate to Spanish: 'the quick brown fox'"
  - *Setup*: Only text-processor agent available
  - *Expected Flow*: Request → Clarification → User Response → Execution

- [ ] **Task 5.1.2**: Test with agents running/not running
  - *Agent States*: Various combinations of available agents
  - *Validation*: Appropriate clarifications for each state

- [ ] **Task 5.1.3**: Validate conversation flow
  - *Flow*: Complete user interaction from start to finish
  - *Context*: Rich conversation history maintained

- [ ] **Task 5.1.4**: Test graph storage completeness
  - *Storage*: All interaction data stored in graph
  - *Retrieval*: Complete context retrievable

#### **Epic 5.2: Integration Testing**
- **User Story**: As a complete system, all components should work together seamlessly for user clarification flows.
- **Acceptance Criteria**:
  - Orchestrator handles clarification flow correctly
  - Planning engine generates good clarification questions
  - Conversation system preserves context
  - Execution proceeds after clarification

**Tasks**:
- [ ] **Task 5.2.1**: Write comprehensive integration test
  - *Test*: End-to-end system integration
  - *Components*: All system components working together

- [ ] **Task 5.2.2**: Test with real AI provider
  - *AI Integration*: Test with actual OpenAI API
  - *Quality*: Validate AI response quality

- [ ] **Task 5.2.3**: Validate performance under load
  - *Performance*: Clarification flow performance
  - *Benchmark*: <100ms additional latency for clarifications

- [ ] **Task 5.2.4**: Test error handling edge cases
  - *Edge Cases*: Various error scenarios
  - *Graceful Degradation*: System handles errors appropriately

---

## **🎯 SUCCESS CRITERIA & VALIDATION**

### **Functional Requirements**:
1. **Capability Gap Detection**: AI correctly identifies missing agent capabilities ✅
2. **Intelligent Clarification**: AI asks specific, contextual questions instead of creating workarounds ✅
3. **Context Preservation**: Full conversation context stored and retrievable ✅
4. **Seamless Flow**: User can respond to clarifications and system proceeds appropriately ✅
5. **Graph Integration**: Complete interaction chain stored in graph database ✅

### **Technical Requirements**:
1. **TDD Compliance**: All code developed using red-green-refactor cycle ✅
2. **Clean Architecture**: Proper domain boundaries maintained ✅
3. **Performance**: Clarification flow adds <100ms to response time ✅
4. **Error Handling**: Graceful degradation when clarification fails ✅
5. **Scalability**: Flow works with multiple concurrent users ✅

### **Example Success Scenario**:
```
User: "Count words and translate to Spanish: 'the quick brown fox'"

AI: "I can count the words for you (I found 4 words in your text), but I notice that no translation agent is currently available. 

Would you like me to:
1. Proceed with just word counting ('the quick brown fox' contains 4 words)
2. Wait for a translation agent to become available
3. Suggest alternative text processing options (character count, text analysis)"

User: "Just do the word counting for now"

AI: "Perfect! The text 'the quick brown fox' contains 4 words."

Graph Storage: Complete interaction chain with reasoning stored
```

---

## **📊 IMPLEMENTATION TIMELINE**

- **Phase 1**: 2-3 hours (Enhanced AI Planning Intelligence)
- **Phase 2**: 2-3 hours (Structured User Clarification Flow)  
- **Phase 3**: 2-3 hours (User Response Processing)
- **Phase 4**: 1-2 hours (Graph Integration & Rich Context)
- **Phase 5**: 1-2 hours (End-to-End Validation)

**Total Estimated Time**: 8-13 hours

---

## **🔧 CURRENT IMPLEMENTATION STATUS**

### **Next Action**: Start Phase 1, Epic 1.1, Task 1.1.1
- **File to Edit**: `internal/planning/application/ai_planning_engine_test.go`
- **Test to Write**: Capability gap detection test
- **Expected**: AI returns CLARIFY when translation requested but unavailable

### **Working Memory - Key Insights**:
1. **Current CLARIFY Handling**: Orchestrator uses `planningResult.Reasoning` as message (line 125)
2. **AI Prompt Location**: `ai_planning_engine.go` system prompt needs capability gap analysis
3. **ExecutionPlan.AgentGap**: Field exists but not effectively used
4. **Graph Relationships**: Cross-domain linking already implemented but needs enrichment

### **Current Code Targets**:
- **orchestrator_service.go:123-125**: CLARIFY handling logic
- **ai_planning_engine.go**: System prompt and capability analysis
- **execution_plan.go**: Reasoning field enhancement
- **conversation.go**: Context preservation for clarifications

---

## **🧠 DEVELOPMENT NOTES & INSIGHTS**

### **Architecture Decisions**:
1. **Leverage Existing CLARIFY Type**: Build on existing `PlanningTypeClarify` infrastructure
2. **Enhance AI Prompts**: Improve capability gap detection in AI prompts rather than adding complex logic
3. **Rich Context Storage**: Store complete AI reasoning in ExecutionPlan.Reasoning field
4. **Graph Relationships**: Use existing cross-domain relationship patterns
5. **User Experience**: Make clarification questions specific and actionable

### **Key Design Patterns**:
1. **TDD First**: Every enhancement starts with failing tests
2. **Clean Architecture**: Maintain domain boundaries throughout
3. **AI-Native**: Let AI handle complexity, provide good prompts and context
4. **Event-Driven**: Use existing event patterns for rich context
5. **Graph-Native**: Store everything in graph for rich queries

### **Critical Success Factors**:
1. **AI Prompt Quality**: Good prompts are crucial for capability gap detection
2. **User Experience**: Clarification questions must be clear and actionable
3. **Context Preservation**: Complete interaction history essential for follow-ups
4. **Performance**: Clarification flow must not significantly impact response time
5. **Integration**: All components must work seamlessly together

---

*This document serves as the working memory and development guide for implementing user-AI interaction with rich context. Update status and insights as development progresses.*
