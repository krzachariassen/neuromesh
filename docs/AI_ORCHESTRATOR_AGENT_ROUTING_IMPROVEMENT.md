# AI Orchestrator Agent Routing Improvement

## Issue Identified
**Date**: July 30, 2025  
**Priority**: High  
**Category**: AI Orchestrator Logic

## Problem Statement

The AI orchestrator is incorrectly routing requests to agents when the request doesn't actually require those agent capabilities.

### Observed Behavior
```
User Input: "Create a comprehensive execution plan for testing a simple flow using the default project"

AI Orchestrator Decision:
- ✅ Planning type: Execute - AI-Native Orchestration requiredAgents=1
- ❌ Routes to: text-processor-001 with action "text-analysis"
- ❌ Intent: "create_execution_plan_for_testing_simple_flow"
```

### Expected Behavior
The orchestrator should recognize that:
1. The request is asking for **execution plan creation** (planning/strategy)
2. This is **NOT** a text analysis task
3. The text-processor agent capabilities are:
   - word-count
   - text-analysis  
   - character-count
4. **None of these capabilities** match the user's request

## Root Cause Analysis

### AI Planning Logic Issues
1. **Capability Mismatch**: The AI is forcing agent usage even when no agent capabilities match
2. **Planning vs Execution Confusion**: Request for "execution plan" is being treated as requiring agent execution
3. **No Agent Gap Detection**: System should detect when NO available agents can handle the request

### Agent Capability Definitions
Current text-processor-001 capabilities:
```go
capabilities := []*proto.AgentCapability{
    {Name: "word-count", Description: "Count the number of words in text"},
    {Name: "text-analysis", Description: "Analyze text properties and characteristics"},  
    {Name: "character-count", Description: "Count the number of characters in text"},
}
```

**None of these handle "execution plan creation"**

## Solution Requirements

### 1. Enhanced Agent Capability Matching
- [ ] Improve AI orchestrator prompt to better match user intent with agent capabilities
- [ ] Add explicit capability gap detection
- [ ] Return "no suitable agent" response when capabilities don't match

### 2. Orchestrator Logic Improvements
```go
// Should detect when no agents can handle the request
if !hasMatchingCapabilities(userIntent, availableAgents) {
    return &OrchestratorResult{
        Success: false,
        Message: "I don't have any agents that can handle this type of request. Available capabilities: word-count, text-analysis, character-count",
        PlanningResult: &PlanningResult{
            Type: planningDomain.PlanningTypeNoCapability,
            AgentGap: []string{"execution-planning", "strategy-creation"},
        },
    }
}
```

### 3. Better AI Prompting
Update orchestrator system prompt to:
- [ ] Explicitly list available agent capabilities
- [ ] Require strict capability matching
- [ ] Detect and report capability gaps
- [ ] Distinguish between planning requests and execution requests

## Test Cases to Validate Fix

### Should NOT Route to Text Processor
```bash
# General conversation
curl -X POST /api/chat -d '{"message": "Hello, how are you?"}'

# Planning requests  
curl -X POST /api/chat -d '{"message": "Create an execution plan for testing"}'

# Strategy requests
curl -X POST /api/chat -d '{"message": "What should I do to improve my workflow?"}'
```

### SHOULD Route to Text Processor
```bash
# Word counting
curl -X POST /api/chat -d '{"message": "Count words in this sentence: Hello world"}'

# Text analysis
curl -X POST /api/chat -d '{"message": "Analyze this text: The quick brown fox"}'

# Character counting  
curl -X POST /api/chat -d '{"message": "How many characters are in: Test message"}'
```

## Implementation Priority

1. **Immediate**: Fix AI orchestrator prompting for capability matching
2. **Short-term**: Add agent gap detection and reporting
3. **Medium-term**: Implement PlanningTypeNoCapability handling
4. **Long-term**: Add more sophisticated capability matching algorithms

## Impact Assessment

### Current Impact
- ❌ Poor user experience (wrong agent responses)
- ❌ Wasted agent resources
- ❌ Incorrect execution flows
- ❌ Confusing system behavior

### Post-Fix Benefits
- ✅ Accurate agent routing
- ✅ Clear "no capability" responses
- ✅ Better resource utilization
- ✅ Improved user trust in system intelligence

## Notes
This issue was discovered during Phase 1 project implementation testing and represents a fundamental flaw in the AI orchestrator's capability matching logic.

---
**Status**: Identified - Needs Implementation  
**Assignee**: Development Team  
**Estimated Effort**: 1-2 days  
**Dependencies**: None
