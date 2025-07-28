package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"neuromesh/internal/graph"
	"neuromesh/internal/logging"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossDomainRelationshipsInNeo4j tests that conversation-execution plan relationships exist
func TestCrossDomainRelationshipsInNeo4j(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Neo4j integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelInfo)

	// Connect to Neo4j - use environment variables or defaults
	neo4jURL := os.Getenv("NEO4J_URI")
	neo4jPassword := os.Getenv("NEO4J_PASSWORD")

	if neo4jURL == "" {
		neo4jURL = "bolt://localhost:7687"
	}
	if neo4jPassword == "" {
		neo4jPassword = "orchestrator123"
	}

	config := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      neo4jURL,
		Neo4jUser:     "neo4j",
		Neo4jPassword: neo4jPassword,
	}

	graphInstance, err := graph.NewNeo4jGraph(ctx, config, logger)
	if err != nil {
		t.Skipf("Cannot connect to Neo4j: %v", err)
	}
	defer graphInstance.Close(ctx)

	t.Run("check what exists in database", func(t *testing.T) {
		// Create a Neo4j session
		session := graphInstance.Driver().NewSession(ctx, neo4j.SessionConfig{})
		defer session.Close(ctx)

		// Query for ANY nodes to see what exists
		query := `MATCH (n) RETURN labels(n), count(*) ORDER BY count(*) DESC`

		result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			var nodes []map[string]interface{}

			records, err := tx.Run(ctx, query, nil)
			if err != nil {
				return nil, err
			}

			for records.Next(ctx) {
				record := records.Record()
				nodes = append(nodes, map[string]interface{}{
					"labels": record.Values[0],
					"count":  record.Values[1],
				})
			}

			return nodes, records.Err()
		})

		require.NoError(t, err, "Failed to query existing nodes")

		nodes := result.([]map[string]interface{})

		if len(nodes) == 0 {
			t.Log("🗃️ Database is completely empty")
		} else {
			t.Log("🗃️ Existing nodes in database:")
			for _, node := range nodes {
				t.Logf("  %v: %v", node["labels"], node["count"])
			}
		}
	})

	t.Run("should find conversation-execution plan relationships", func(t *testing.T) {
		// Create a Neo4j session
		session := graphInstance.Driver().NewSession(ctx, neo4j.SessionConfig{})
		defer session.Close(ctx)

		// Query for relationships between conversations and execution plans
		query := `
			MATCH (c:Conversation)-[r:LINKED_TO_PLAN]->(p:execution_plan)
			RETURN c.id as conversation_id, p.id as plan_id, type(r) as relationship_type
			LIMIT 10
		`

		result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			var relationships []map[string]interface{}

			records, err := tx.Run(ctx, query, nil)
			if err != nil {
				return nil, err
			}

			for records.Next(ctx) {
				record := records.Record()
				relationships = append(relationships, map[string]interface{}{
					"conversation_id":   record.Values[0],
					"plan_id":           record.Values[1],
					"relationship_type": record.Values[2],
				})
			}

			return relationships, records.Err()
		})

		require.NoError(t, err, "Failed to query conversation-execution plan relationships")

		relationships := result.([]map[string]interface{})

		// Log what we found
		if len(relationships) > 0 {
			t.Logf("✅ Found %d conversation-execution plan relationships:", len(relationships))
			for i, rel := range relationships {
				t.Logf("  %d. Conversation %s -> %s -> ExecutionPlan %s",
					i+1, rel["conversation_id"], rel["relationship_type"], rel["plan_id"])
			}
		} else {
			t.Log("❌ No conversation-execution plan relationships found")
		}

		// The relationships should exist if our orchestrator is working correctly
		assert.Greater(t, len(relationships), 0, "Should have conversation-execution plan relationships")
	})

	t.Run("should find all expected node types", func(t *testing.T) {
		// Create a Neo4j session
		session := graphInstance.Driver().NewSession(ctx, neo4j.SessionConfig{})
		defer session.Close(ctx)

		// Query for all the node types we expect to see
		nodeTypes := []string{"User", "Session", "Conversation", "ConversationMessage", "execution_plan", "execution_step", "Decision", "Analysis"}

		for _, nodeType := range nodeTypes {
			query := fmt.Sprintf("MATCH (n:%s) RETURN count(n) as count", nodeType)

			result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
				record, err := tx.Run(ctx, query, nil)
				if err != nil {
					return int64(0), err
				}

				if record.Next(ctx) {
					return record.Record().Values[0], nil
				}
				return int64(0), record.Err()
			})

			require.NoError(t, err, "Failed to query %s nodes", nodeType)

			count := result.(int64)
			t.Logf("Found %d %s nodes", count, nodeType)
		}
	})

	t.Run("should show complete relationship graph", func(t *testing.T) {
		// Create a Neo4j session
		session := graphInstance.Driver().NewSession(ctx, neo4j.SessionConfig{})
		defer session.Close(ctx)

		// Query for all relationships to see the complete picture
		query := `
			MATCH (n)-[r]->(m)
			RETURN labels(n)[0] as from_type, type(r) as relationship, labels(m)[0] as to_type, count(*) as count
			ORDER BY count DESC
		`

		result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			var relationships []map[string]interface{}

			records, err := tx.Run(ctx, query, nil)
			if err != nil {
				return nil, err
			}

			for records.Next(ctx) {
				record := records.Record()
				relationships = append(relationships, map[string]interface{}{
					"from_type":    record.Values[0],
					"relationship": record.Values[1],
					"to_type":      record.Values[2],
					"count":        record.Values[3],
				})
			}

			return relationships, records.Err()
		})

		require.NoError(t, err, "Failed to query all relationships")

		relationships := result.([]map[string]interface{})

		t.Log("📊 Complete relationship overview:")
		for _, rel := range relationships {
			t.Logf("  %s -[%s]-> %s (%v relationships)",
				rel["from_type"], rel["relationship"], rel["to_type"], rel["count"])
		}
	})
}
