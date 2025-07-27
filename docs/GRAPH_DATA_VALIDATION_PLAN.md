# Graph Data Validation Plan

## Overview
Before building the API and UI to display graph data, we need to validate that the orchestrator is correctly storing all aspects of the execution flow in Neo4j. This ensures we understand the actual data structure and relationships before implementing the visualization.

## Validation Steps

### Phase 1: End-to-End Orchestration Test
1. **Start NeuroMesh Server**
   - Start the unified API server with Neo4j connection
   - Verify RabbitMQ and Neo4j connections are established

2. **Start Text Processor Agent**
   - Start the `agents/text-processor` agent
   - Verify agent registration in the system
   - Confirm agent is listening on RabbitMQ queues

3. **Execute Test Request**
   - Use the orchestrator API to request: "Count words in the string 'Hello world this is a test'"
   - This should trigger:
     - User request creation
     - AI decision making
     - Agent task delegation
     - Execution plan creation
     - Task execution
     - Result synthesis

### Phase 2: Neo4j Data Inspection
4. **Direct Database Query**
   - Connect to Neo4j browser/console
   - Query all nodes: `MATCH (n) RETURN n`
   - Query all relationships: `MATCH (n)-[r]->(m) RETURN n, r, m`
   - Verify complete execution flow is stored

5. **Validate Node Types**
   - User nodes
   - Conversation nodes  
   - Agent nodes
   - Execution plan nodes
   - Execution step nodes
   - Result nodes
   - Message nodes

6. **Validate Relationship Types**
   - User → Conversation (participates_in)
   - Conversation → Agent (assigned_to)
   - Agent → ExecutionPlan (creates)
   - ExecutionPlan → ExecutionStep (contains)
   - ExecutionStep → Result (produces)
   - Agent → Agent (delegates_to)

### Phase 3: Schema Validation
7. **Node Properties Verification**
   ```cypher
   // Check User node properties
   MATCH (u:User) RETURN properties(u)
   
   // Check Conversation node properties
   MATCH (c:Conversation) RETURN properties(c)
   
   // Check Agent node properties
   MATCH (a:Agent) RETURN properties(a)
   
   // Check ExecutionPlan properties
   MATCH (ep:ExecutionPlan) RETURN properties(ep)
   
   // Check ExecutionStep properties
   MATCH (es:ExecutionStep) RETURN properties(es)
   
   // Check Result properties
   MATCH (r:Result) RETURN properties(r)
   ```

8. **Relationship Properties Verification**
   ```cypher
   // Check all relationship types and properties
   MATCH ()-[r]->() RETURN type(r), properties(r)
   ```

### Phase 4: Data Completeness Check
9. **Verify Complete Flow**
   - Ensure the word count request creates a complete graph path
   - Check that all execution steps are linked
   - Verify timestamps and status updates
   - Confirm result data is properly stored

10. **Identify Missing Data**
    - Document any missing nodes or relationships
    - Note any incomplete property sets
    - Identify schema improvements needed

## Expected Neo4j Graph Structure

```
(User)-[:INITIATES]->(Conversation)
(Conversation)-[:ASSIGNED_TO]->(Agent:Orchestrator)
(Agent:Orchestrator)-[:DELEGATES_TO]->(Agent:TextProcessor)
(Agent:TextProcessor)-[:CREATES]->(ExecutionPlan)
(ExecutionPlan)-[:CONTAINS]->(ExecutionStep:WordCount)
(ExecutionStep)-[:PRODUCES]->(Result)
(Result)-[:SYNTHESIZED_BY]->(Agent:Orchestrator)
```

## Success Criteria
- [ ] Complete execution flow stored in Neo4j
- [ ] All node types present with required properties
- [ ] All relationships correctly established
- [ ] Timestamps and status tracking working
- [ ] Result data properly structured
- [ ] No orphaned nodes or broken relationships

## Next Steps After Validation
Once we confirm the graph data is complete and correctly structured:

1. **Update API Implementation**
   - Replace mock data with real Neo4j queries
   - Implement proper graph traversal queries
   - Add error handling for missing data

2. **Enhance Graph Schema**
   - Add any missing node types discovered
   - Improve property schemas
   - Add indexes for performance

3. **Build Real Graph Visualization**
   - Connect React UI to real graph data
   - Implement dynamic positioning algorithms
   - Add real-time updates for active executions

## Tools for Validation
- **Neo4j Browser**: http://localhost:7474
- **Cypher Queries**: Direct database inspection
- **API Endpoints**: 
  - POST `/api/chat` - Submit orchestration request
  - GET `/api/v1/conversations/{id}/graph` - Retrieve graph data
- **Agent Logs**: Monitor execution flow
- **RabbitMQ Management**: Monitor message flow

## Timeline
- **Day 1**: Execute test, inspect Neo4j data, document findings
- **Day 2**: Fix any data storage issues, implement real API
- **Day 3**: Connect UI to real data, test complete flow

---

**Note**: This validation step is crucial to ensure we're building the visualization on top of a solid, complete data foundation rather than assumptions about what should be stored.
