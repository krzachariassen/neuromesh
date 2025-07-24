# Dynamic Orchestrator Graph Schema Design

## Overview

This document defines the graph schema for AI-native orchestration that supports dynamic planning, agent-to-agent coordination, and real-time plan adaptation. This schema enables the transition from static text-based planning to structured, graph-native orchestration.

**CRITICAL**: This schema is designed to be truly graph-native, using relationships between nodes instead of foreign key references. It integrates seamlessly with our existing agent graph model.

## Core Principles

1. **Graph-Native**: All relationships are edges, no foreign key IDs
2. **Dynamic Planning**: Plans can be modified during execution via graph relationships
3. **Agent Integration**: Leverages existing agent nodes and capabilities
4. **Audit Trail**: Complete traceability through relationship chains
5. **Human-in-Loop**: Approval gates via decision relationships

## Node Types

### 1. Analysis (Planning Domain)
**Purpose**: AI analysis of user requests - connects to existing Message nodes
```cypher
(:Analysis {
  id: String,              // Unique identifier
  intent: String,          // What user wants to accomplish
  category: String,        // Domain area (deployment, security, etc.)
  confidence: Integer,     // 0-100 confidence score
  reasoning: String,       // AI reasoning for this analysis
  timestamp: DateTime,     // When analysis was created
  status: String          // DRAFT, COMPLETED, SUPERSEDED
})
```

### 2. ExecutionPlan (Planning Domain)
**Purpose**: Structured plan - connected to Analysis and Steps via relationships
```cypher
(:ExecutionPlan {
  id: String,              // Unique identifier
  name: String,            // Human-readable plan name
  description: String,     // Plan overview
  status: String,          // DRAFT, APPROVED, EXECUTING, COMPLETED, FAILED
  created_at: DateTime,    // When plan was created
  approved_at: DateTime,   // When plan was approved (if applicable)
  started_at: DateTime,    // When execution started
  completed_at: DateTime,  // When execution completed
  estimated_duration: Integer, // Estimated duration in minutes
  actual_duration: Integer,    // Actual duration in minutes
  can_modify: Boolean,     // Can this plan be modified during execution?
  priority: String         // LOW, MEDIUM, HIGH, CRITICAL
})
```

### 3. ExecutionStep (Planning Domain)
**Purpose**: Individual steps - connected to Plan and Agent nodes via relationships
```cypher
(:ExecutionStep {
  id: String,              // Unique identifier
  step_number: Integer,    // Execution order within plan
  name: String,            // Step name
  description: String,     // What this step does
  status: String,          // PENDING, ASSIGNED, EXECUTING, COMPLETED, FAILED, SKIPPED
  estimated_duration: Integer, // Estimated duration in minutes
  actual_duration: Integer,    // Actual duration in minutes
  inputs: String,          // JSON of input parameters
  outputs: String,         // JSON of output results
  error_message: String,   // Error details if failed
  can_modify: Boolean,     // Can this step be modified during execution?
  is_critical: Boolean,    // Is this step critical to overall success?
  retry_count: Integer,    // Number of times this step has been retried
  max_retries: Integer,    // Maximum allowed retries
  started_at: DateTime,    // When step execution started
  completed_at: DateTime   // When step execution completed
})
```

### 4. Decision (Orchestrator Domain)
**Purpose**: AI decisions - connected to Messages, Analysis, Plans via relationships
```cypher
(:Decision {
  id: String,              // Unique identifier
  type: String,            // CLARIFY, EXECUTE, MODIFY_PLAN, ESCALATE
  action: String,          // Specific action to take
  parameters: String,      // JSON of decision parameters
  clarification_question: String, // Question to ask user (if CLARIFY)
  reasoning: String,       // AI reasoning for this decision
  confidence: Integer,     // 0-100 confidence in this decision
  timestamp: DateTime,     // When decision was made
  status: String,          // PENDING, APPROVED, REJECTED, EXECUTED
  requires_approval: Boolean // Does this decision need human approval?
})
```

### 5. AgentCommunication (New)
**Purpose**: Agent-to-agent messages - connected to Agent nodes via relationships
```cypher
(:AgentCommunication {
  id: String,              // Unique identifier
  message_type: String,    // REQUEST, RESPONSE, NOTIFICATION, ESCALATION
  subject: String,         // Communication subject
  content: String,         // Message content
  priority: String,        // LOW, MEDIUM, HIGH, URGENT
  status: String,          // SENT, RECEIVED, ACKNOWLEDGED, RESPONDED
  response_content: String, // Response message (if applicable)
  timestamp: DateTime,     // When message was sent
  response_timestamp: DateTime, // When response was received
  requires_action: Boolean, // Does this require recipient action?
  action_taken: String     // Description of action taken
})
```

### 6. PlanModification (New)
**Purpose**: Track plan changes - connected to Plans/Steps/Agents via relationships
```cypher
(:PlanModification {
  id: String,              // Unique identifier
  modification_type: String, // ADD_STEP, REMOVE_STEP, MODIFY_STEP, REASSIGN_AGENT, CHANGE_ORDER
  description: String,     // Human-readable description of change
  old_value: String,       // JSON of previous state
  new_value: String,       // JSON of new state
  reason: String,          // Why this modification was made
  timestamp: DateTime,     // When modification was made
  status: String,          // REQUESTED, APPROVED, REJECTED, APPLIED
  impact_assessment: String // Assessment of change impact
})
```

### 7. ExecutionContext (New)
**Purpose**: Environment state - connected to Plans via relationships
```cypher
(:ExecutionContext {
  id: String,              // Unique identifier
  environment: String,     // DEVELOPMENT, STAGING, PRODUCTION
  variables: String,       // JSON of environment variables
  constraints: String,     // JSON of execution constraints
  resources: String,       // JSON of available resources
  policies: String,        // JSON of applicable policies
  permissions: String,     // JSON of execution permissions
  created_at: DateTime,    // When context was created
  updated_at: DateTime,    // When context was last updated
  is_active: Boolean       // Is this context currently active?
})
```

## Relationships (Graph-Native Design)

### Core Planning Flow
```cypher
// User request triggers analysis
(:Message)-[:TRIGGERS_ANALYSIS]->(:Analysis)

// Analysis creates execution plan
(:Analysis)-[:CREATES_PLAN]->(:ExecutionPlan)

// Plan contains ordered steps
(:ExecutionPlan)-[:CONTAINS_STEP {order: step_number}]->(:ExecutionStep)

// Steps can depend on other steps
(:ExecutionStep)-[:DEPENDS_ON]->(:ExecutionStep)
(:ExecutionStep)-[:BLOCKS]->(:ExecutionStep)

// Analysis informs decisions
(:Analysis)-[:INFORMS_DECISION]->(:Decision)
```

### Agent Assignment and Execution
```cypher
// Steps are assigned to agents (uses existing Agent nodes)
(:ExecutionStep)-[:ASSIGNED_TO]->(:Agent)

// Plans are managed by agents
(:ExecutionPlan)-[:MANAGED_BY]->(:Agent)

// Steps require specific agent capabilities
(:ExecutionStep)-[:REQUIRES_CAPABILITY]->(:Capability)

// Agents execute steps and produce results
(:Agent)-[:EXECUTES]->(:ExecutionStep)
(:ExecutionStep)-[:PRODUCES_RESULT]->(:ExecutionStep) // Next step
```

### Decision Making and Approval
```cypher
// Decisions relate to specific requests and plans
(:Decision)-[:DECIDES_ON]->(:Message)
(:Decision)-[:AFFECTS_PLAN]->(:ExecutionPlan)
(:Decision)-[:AFFECTS_STEP]->(:ExecutionStep)

// Users approve decisions
(:User)-[:APPROVES]->(:Decision)
(:User)-[:REJECTS]->(:Decision)

// Decisions escalate to users when needed
(:Decision)-[:ESCALATES_TO]->(:User)
```

### Agent Communication (Graph-Native)
```cypher
// Agents send and receive communications
(:Agent)-[:SENDS]->(:AgentCommunication)
(:Agent)-[:RECEIVES]->(:AgentCommunication)

// Communications relate to plans and steps
(:AgentCommunication)-[:ABOUT_PLAN]->(:ExecutionPlan)
(:AgentCommunication)-[:ABOUT_STEP]->(:ExecutionStep)

// Communications can be responses to other communications
(:AgentCommunication)-[:RESPONDS_TO]->(:AgentCommunication)
```

### Plan Modification Tracking
```cypher
// Modifications target specific plans or steps
(:PlanModification)-[:MODIFIES_PLAN]->(:ExecutionPlan)
(:PlanModification)-[:MODIFIES_STEP]->(:ExecutionStep)

// Agents request and approve modifications
(:Agent)-[:REQUESTS_MODIFICATION]->(:PlanModification)
(:Agent)-[:APPROVES_MODIFICATION]->(:PlanModification)
(:User)-[:APPROVES_MODIFICATION]->(:PlanModification)

// Modifications can create new steps or modify existing ones
(:PlanModification)-[:CREATES_STEP]->(:ExecutionStep)
(:PlanModification)-[:REMOVES_STEP]->(:ExecutionStep)
```

### Context and Environment
```cypher
// Plans execute within contexts
(:ExecutionPlan)-[:EXECUTES_IN]->(:ExecutionContext)

// Steps may require specific context
(:ExecutionStep)-[:REQUIRES_CONTEXT]->(:ExecutionContext)

// Agents have access to contexts
(:Agent)-[:HAS_ACCESS_TO]->(:ExecutionContext)
```

### Complete Audit Trail (Chain of Relationships)
```cypher
// Full traceability chain
(:User)-[:INITIATES]->(:Message)
  -[:TRIGGERS_ANALYSIS]->(:Analysis)
  -[:CREATES_PLAN]->(:ExecutionPlan)
  -[:CONTAINS_STEP]->(:ExecutionStep)
  -[:ASSIGNED_TO]->(:Agent)

// Execution results flow back
(:Agent)-[:EXECUTES]->(:ExecutionStep)
  -[:PRODUCES_RESULT]->(:ExecutionStep)
  -[:UPDATES_PLAN]->(:ExecutionPlan)
  -[:INFORMS_ANALYSIS]->(:Analysis)
```

### Integration with Existing Graph Model
```cypher
// Leverages existing User/Session/Conversation structure
(:User)-[:HAS_SESSION]->(:Session)
  -[:CONTAINS_CONVERSATION]->(:Conversation)
  -[:CONTAINS_MESSAGE]->(:Message)
  -[:TRIGGERS_ANALYSIS]->(:Analysis) // New integration point

// Leverages existing Agent/Capability structure  
(:Agent)-[:HAS_CAPABILITY]->(:Capability) // Existing
(:ExecutionStep)-[:REQUIRES_CAPABILITY]->(:Capability) // New usage
(:ExecutionStep)-[:ASSIGNED_TO]->(:Agent) // New assignment
```

## Example Query Patterns (Graph-Native)

### 1. Get Complete Plan with All Steps and Assigned Agents
```cypher
MATCH (a:Analysis {id: $analysisId})-[:CREATES_PLAN]->(p:ExecutionPlan)
MATCH (p)-[:CONTAINS_STEP]->(s:ExecutionStep)
OPTIONAL MATCH (s)-[:ASSIGNED_TO]->(agent:Agent)
OPTIONAL MATCH (s)-[:REQUIRES_CAPABILITY]->(cap:Capability)
RETURN p, s, agent, cap
ORDER BY s.step_number
```

### 2. Find All Plan Modifications with Responsible Agents
```cypher
MATCH (p:ExecutionPlan {id: $planId})
MATCH (m:PlanModification)-[:MODIFIES_PLAN]->(p)
OPTIONAL MATCH (reqAgent:Agent)-[:REQUESTS_MODIFICATION]->(m)
OPTIONAL MATCH (appAgent:Agent)-[:APPROVES_MODIFICATION]->(m)
RETURN m, reqAgent.name, appAgent.name
ORDER BY m.timestamp DESC
```

### 3. Get Agent Communication Chain for a Plan
```cypher
MATCH (p:ExecutionPlan {id: $planId})
MATCH (comm:AgentCommunication)-[:ABOUT_PLAN]->(p)
MATCH (fromAgent:Agent)-[:SENDS]->(comm)
MATCH (toAgent:Agent)-[:RECEIVES]->(comm)
OPTIONAL MATCH (comm)-[:RESPONDS_TO]->(parentComm:AgentCommunication)
RETURN comm, fromAgent.name, toAgent.name, parentComm
ORDER BY comm.timestamp
```

### 4. Find Blocked Steps and Their Dependencies
```cypher
MATCH (step:ExecutionStep {status: 'PENDING'})
MATCH (step)-[:DEPENDS_ON]->(blockingStep:ExecutionStep)
WHERE blockingStep.status NOT IN ['COMPLETED', 'SKIPPED']
MATCH (blockingStep)-[:ASSIGNED_TO]->(agent:Agent)
RETURN step.name, blockingStep.name, agent.name, blockingStep.status
```

### 5. Get Agent Workload (Current Assignments)
```cypher
MATCH (agent:Agent {id: $agentId})
MATCH (agent)<-[:ASSIGNED_TO]-(step:ExecutionStep)
MATCH (step)<-[:CONTAINS_STEP]-(plan:ExecutionPlan)
WHERE step.status IN ['PENDING', 'ASSIGNED', 'EXECUTING']
RETURN plan.name, step.name, step.status, step.estimated_duration
ORDER BY plan.priority DESC, step.step_number
```

### 6. Find Plans Requiring Approval
```cypher
MATCH (p:ExecutionPlan {status: 'DRAFT'})
MATCH (d:Decision)-[:AFFECTS_PLAN]->(p)
WHERE d.requires_approval = true AND NOT EXISTS((u:User)-[:APPROVES]->(d))
OPTIONAL MATCH (d)-[:ESCALATES_TO]->(u:User)
RETURN p, d, u.name
```

### 7. Get Full Audit Trail for a Request
```cypher
MATCH path = (u:User)-[:INITIATES]->(msg:Message)
  -[:TRIGGERS_ANALYSIS]->(a:Analysis)
  -[:CREATES_PLAN]->(p:ExecutionPlan)
  -[:CONTAINS_STEP]->(s:ExecutionStep)
  -[:ASSIGNED_TO]->(agent:Agent)
WHERE msg.id = $messageId
RETURN path
```

### 8. Find Available Agents for a Capability
```cypher
MATCH (step:ExecutionStep {id: $stepId})
MATCH (step)-[:REQUIRES_CAPABILITY]->(cap:Capability)
MATCH (agent:Agent)-[:HAS_CAPABILITY]->(cap)
WHERE agent.status = 'ACTIVE' 
  AND NOT EXISTS((agent)<-[:ASSIGNED_TO]-(:ExecutionStep {status: 'EXECUTING'}))
RETURN agent, cap
```

### 9. Get Communication Thread Between Two Agents
```cypher
MATCH (agent1:Agent {id: $agent1Id})
MATCH (agent2:Agent {id: $agent2Id})
MATCH path = (agent1)-[:SENDS|RECEIVES*]-(comm:AgentCommunication)-[:SENDS|RECEIVES*]-(agent2)
WHERE comm.timestamp >= $startDate AND comm.timestamp <= $endDate
RETURN path
ORDER BY comm.timestamp
```

### 10. Find Plans That Can Be Modified (Dynamic Planning)
```cypher
MATCH (p:ExecutionPlan)
WHERE p.can_modify = true 
  AND p.status IN ['EXECUTING', 'DRAFT']
MATCH (p)-[:CONTAINS_STEP]->(s:ExecutionStep)
WHERE s.can_modify = true
OPTIONAL MATCH (p)-[:MANAGED_BY]->(manager:Agent)
RETURN p, collect(s), manager
```

## MVP Implementation Scope (Anti-Overengineering)

### What We're Actually Solving
**Problem**: Current `Decision` domain has unstructured strings:
```go
type Decision struct {
    ExecutionPlan     string  // ❌ "Deploy app using kubectl and check status"
    AgentCoordination string  // ❌ "kubernetes-agent handles deployment, monitoring-agent tracks"
}
```

**Solution**: Replace with structured graph data that can be queried and executed.

### MVP Scope (Phase 1 Only)
**Implement ONLY these 2 node types:**

#### 1. ExecutionPlan (Planning Domain)
```cypher
(:ExecutionPlan {
  id: String,
  name: String,
  description: String,
  status: String,          // DRAFT, APPROVED, EXECUTING, COMPLETED, FAILED
  created_at: DateTime,
  priority: String         // LOW, MEDIUM, HIGH, CRITICAL
})
```

#### 2. ExecutionStep (Planning Domain)  
```cypher
(:ExecutionStep {
  id: String,
  step_number: Integer,    // Order in plan
  name: String,
  description: String,
  status: String,          // PENDING, ASSIGNED, EXECUTING, COMPLETED, FAILED
  inputs: String,          // JSON parameters
  outputs: String          // JSON results
})
```

### MVP Relationships (Only These 3)
```cypher
// 1. Analysis creates plan (replace string field)
(:Analysis)-[:CREATES_PLAN]->(:ExecutionPlan)

// 2. Plan contains ordered steps (replace string planning)
(:ExecutionPlan)-[:CONTAINS_STEP {order: step_number}]->(:ExecutionStep)

// 3. Steps assigned to agents (replace string coordination)
(:ExecutionStep)-[:ASSIGNED_TO]->(:Agent)
```

### Implementation Effort
- **ExecutionPlan domain model**: ~100 lines (copy Analysis pattern)
- **ExecutionStep domain model**: ~100 lines  
- **GraphExecutionPlanRepository**: ~200 lines (copy GraphAnalysisRepository)
- **Tests**: ~200 lines (copy existing test patterns)
- **Integration**: ~100 lines (update Decision domain to use graph relationships)

**Total: ~700 lines, 2-3 days following TDD**

### What This Achieves
1. ✅ **Structured Planning**: Replace string-based plans with queryable graph data
2. ✅ **Agent Assignment**: Clear agent responsibilities via relationships  
3. ✅ **Basic Orchestration**: Plans → Steps → Agents execution flow
4. ✅ **Graph Queries**: Find plans by agent, steps by status, etc.
5. ✅ **Foundation**: Base for future dynamic features (if needed)

### What We Skip (Anti-Overengineering)
- ❌ Agent-to-agent communication (not needed yet)
- ❌ Plan modification tracking (not needed yet)  
- ❌ Complex approval workflows (not needed yet)
- ❌ Environment contexts (not needed yet)
- ❌ Dynamic plan evolution (not needed yet)

**Principle**: Implement the minimal structure that replaces string-based planning. Add complexity only when we have proven need.
