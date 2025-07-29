# Graph-Native Architecture Cleanup

## Overview
Remove embedded foreign keys from domain entities and implement proper graph-native relationships for conversations, projects, sessions, and users.

## Problem Statement

### Current Architecture Issues

#### 1. Embedded Foreign Keys Anti-Pattern
**Conversation Entity** contains embedded foreign keys:
```go
type Conversation struct {
    ProjectID       string   // Should be graph relationship
    SessionID       string   // Should be graph relationship  
    ExecutionPlanIDs []string // Should be graph relationship
    UserID          string   // Should be graph relationship
}
```

#### 2. Missing Domain Entities
- **Project**: Referenced as strings but no actual Project nodes exist
- **Session**: Exists but not properly connected to project/conversation graph

#### 3. Incomplete Relationship Graph
Current state:
```
user ← → session (disconnected islands)
conversation (with embedded foreign keys)
```

Desired state:
```
user → session → project
  ↓      ↓        ↓
conversation ← execution_plan
```

## Graph-Native Solution

### 1. Remove Embedded Foreign Keys

#### Before (Anti-Pattern):
```go
type Conversation struct {
    ID               string
    ProjectID        string   // ❌ Embedded foreign key
    SessionID        string   // ❌ Embedded foreign key
    UserID           string   // ❌ Embedded foreign key
    ExecutionPlanIDs []string // ❌ Embedded foreign key array
    // ... other properties
}
```

#### After (Graph-Native):
```go
type Conversation struct {
    ID          string
    Content     string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    // All relationships handled by graph edges
}
```

### 2. Create Missing Project Domain Entity

```go
// internal/project/domain/project.go
type Project struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    TenantID    string    `json:"tenant_id"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    Status      ProjectStatus `json:"status"`
}

type ProjectStatus string
const (
    ProjectStatusActive   ProjectStatus = "ACTIVE"
    ProjectStatusArchived ProjectStatus = "ARCHIVED"
)
```

### 3. Establish Graph-Native Relationships

#### Core Relationship Graph:
```cypher
// User-Session-Project hierarchy
(user:User)-[:HAS_SESSION]->(session:Session)
(session:Session)-[:BELONGS_TO]->(project:Project)

// Conversation relationships  
(conversation:Conversation)-[:IN_SESSION]->(session:Session)
(conversation:Conversation)-[:BELONGS_TO]->(project:Project)
(conversation:Conversation)-[:CREATED_BY]->(user:User)

// Execution relationships
(conversation:Conversation)-[:HAS_EXECUTION_PLAN]->(execution_plan:ExecutionPlan)
(execution_plan:ExecutionPlan)-[:HAS_STEP]->(execution_step:ExecutionStep)
(execution_step:ExecutionStep)-[:HAS_RESULT]->(agent_result:AgentResult)
```

## Implementation Plan

### Phase 1: Create Project Domain
- [ ] Create `internal/project/domain/project.go`
- [ ] Create `ProjectRepository` interface
- [ ] Implement graph-based project repository
- [ ] Add project creation/management endpoints

### Phase 2: Update Conversation Domain
- [ ] Remove embedded foreign keys from `Conversation` struct
- [ ] Update conversation repository to use graph relationships
- [ ] Modify conversation creation to establish proper relationships

### Phase 3: Fix Session Relationships
- [ ] Update session creation to link to projects
- [ ] Establish session-conversation relationships
- [ ] Remove embedded session references

### Phase 4: Repository Layer Updates
- [ ] Update `ConversationRepository` to query via relationships
- [ ] Add relationship-based query methods:
   - `GetConversationsByProject(projectID)`
   - `GetConversationsBySession(sessionID)`
   - `GetConversationsByUser(userID)`

### Phase 5: API Layer Updates
- [ ] Update BFF endpoints to work with graph relationships
- [ ] Modify chat API to create proper relationship chains
- [ ] Update conversation queries to use graph traversal

## Graph Query Examples

### Create Complete Relationship Chain
```cypher
// When creating a new conversation
MATCH (u:User {id: $userId})
MATCH (s:Session {id: $sessionId})
MATCH (p:Project {id: $projectId})
CREATE (c:Conversation {
    id: $conversationId,
    content: $content,
    created_at: datetime(),
    updated_at: datetime()
})
CREATE (c)-[:CREATED_BY]->(u)
CREATE (c)-[:IN_SESSION]->(s)  
CREATE (c)-[:BELONGS_TO]->(p)
RETURN c
```

### Query Conversations by Project
```cypher
// Get all conversations in a project
MATCH (p:Project {id: $projectId})<-[:BELONGS_TO]-(c:Conversation)
RETURN c ORDER BY c.created_at DESC
```

### Query Complete Context
```cypher
// Get conversation with full context
MATCH (c:Conversation {id: $conversationId})
MATCH (c)-[:CREATED_BY]->(u:User)
MATCH (c)-[:IN_SESSION]->(s:Session)
MATCH (c)-[:BELONGS_TO]->(p:Project)
OPTIONAL MATCH (c)-[:HAS_EXECUTION_PLAN]->(ep:ExecutionPlan)
RETURN c, u, s, p, collect(ep) as execution_plans
```

## Migration Strategy

### Data Migration
```cypher
// Create missing project nodes
CREATE (p:Project {
    id: "default-project",
    name: "Default Project", 
    description: "Default project for existing conversations",
    tenant_id: "default-tenant",
    created_at: datetime(),
    status: "ACTIVE"
})

// Migrate conversation relationships
MATCH (c:Conversation)
MATCH (p:Project {id: c.project_id})
MATCH (u:User {id: c.user_id})
MATCH (s:Session {id: c.session_id})
CREATE (c)-[:BELONGS_TO]->(p)
CREATE (c)-[:CREATED_BY]->(u)  
CREATE (c)-[:IN_SESSION]->(s)

// Remove embedded foreign keys
MATCH (c:Conversation)
REMOVE c.project_id, c.user_id, c.session_id, c.execution_plan_ids

// Link sessions to projects
MATCH (s:Session)-[:CONTAINS]->(c:Conversation)-[:BELONGS_TO]->(p:Project)
CREATE (s)-[:BELONGS_TO]->(p)
```

## Benefits

### Technical Benefits
- **True Graph Database Usage**: Leverages Neo4j's strength in relationships
- **Query Flexibility**: Easy traversal of relationship chains
- **Data Integrity**: Referential integrity enforced by graph structure
- **Performance**: Optimized graph queries vs embedded array lookups

### Architecture Benefits  
- **Clean Domain Models**: Entities focus on core properties
- **Separation of Concerns**: Relationships managed by graph layer
- **Scalability**: Relationship queries scale better than embedded arrays
- **Maintainability**: Changes to relationships don't require entity updates

### Development Benefits
- **Intuitive Queries**: Natural graph traversal patterns
- **Rich Context**: Easy to query related entities
- **Flexibility**: Add new relationships without schema changes
- **Debugging**: Visual relationship exploration in Neo4j browser

## Testing Strategy

### Unit Tests
- [ ] Test entity creation without embedded foreign keys
- [ ] Test repository relationship operations
- [ ] Test migration scripts with sample data

### Integration Tests  
- [ ] Test end-to-end conversation creation with relationships
- [ ] Test query performance with relationship traversal
- [ ] Test data consistency after migration

### Graph Tests
- [ ] Verify relationship integrity
- [ ] Test orphaned node detection
- [ ] Validate migration completeness

## Success Criteria

- [ ] No embedded foreign keys in any domain entities
- [ ] All relationships represented as graph edges
- [ ] Project domain entity exists and functions
- [ ] Complete session-project-conversation relationship chain
- [ ] All queries use graph traversal instead of embedded lookups
- [ ] Migration completed without data loss
- [ ] Performance maintained or improved

## Priority: High
This addresses fundamental architectural issues that violate graph database principles and clean architecture patterns.

---

**Created**: July 29, 2025  
**Status**: Planned  
**Estimated Effort**: 2-3 sprints  
**Dependencies**: None  
**Impact**: High architectural improvement, significant implementation effort
