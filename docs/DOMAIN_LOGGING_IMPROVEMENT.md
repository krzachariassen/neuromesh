# Domain-Specific Logging Improvement

## Overview
Add domain identifiers to all log messages to improve debugging and system observability by clearly identifying which domain/component is generating each log entry.

## Problem Statement
Currently, log messages don't clearly indicate which domain or component is generating them, making it difficult to:
- Debug issues across domains
- Understand system flow and behavior
- Filter logs by specific domains during troubleshooting

## Proposed Solution

### Domain Identifier Structure
Each log message should include a domain identifier following this pattern:
```
[DOMAIN:COMPONENT] Log message content
```

### Domain Categories
- **CONVERSATION**: User conversations, messages, chat handling
- **EXECUTION**: Agent execution, synthesis, results
- **PLANNING**: AI planning, execution plan creation
- **AGENT**: Agent registration, discovery, lifecycle
- **ORCHESTRATION**: Service coordination, workflow management
- **GRAPH**: Neo4j operations, graph persistence
- **MESSAGING**: Message bus, communication between components
- **USER**: User management, sessions, authentication
- **PROJECT**: Project management, member handling

### Implementation Approach (TDD)

1. **RED Phase**: Create failing tests that expect domain identifiers in log messages
2. **GREEN Phase**: Implement minimal domain logging wrapper
3. **REFACTOR Phase**: Apply domain logging throughout all components

### Example Implementation

```go
// Domain-aware logger wrapper
type DomainLogger struct {
    logger logging.Logger
    domain string
    component string
}

func (dl *DomainLogger) Info(msg string, fields ...interface{}) {
    domainMsg := fmt.Sprintf("[%s:%s] %s", dl.domain, dl.component, msg)
    dl.logger.Info(domainMsg, fields...)
}

// Usage in ExecutionCoordinator
logger := NewDomainLogger(baseLogger, "EXECUTION", "COORDINATOR")
logger.Info("Synthesis result stored successfully", "planID", planID)
// Output: [EXECUTION:COORDINATOR] Synthesis result stored successfully planID=plan-123
```

### Benefits
- **Improved Debugging**: Quickly identify log source domain
- **Better Monitoring**: Filter and analyze logs by domain
- **System Understanding**: Clear view of inter-domain communication
- **Production Support**: Faster issue resolution with domain context

### Implementation Steps
1. Create `DomainLogger` wrapper using TDD approach
2. Update each domain package to use domain-specific loggers
3. Add domain identifiers to existing log statements
4. Validate with integration tests

### Priority: Medium
This improvement enhances observability without affecting core functionality. Can be implemented incrementally domain by domain.

## Related Issues
- Better debugging experience requested by user
- Need for clearer system observability
- Production troubleshooting improvements
