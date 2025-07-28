package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"neuromesh/internal/graph"
	"neuromesh/internal/logging"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	// Initialize logger
	logger := logging.NewStructuredLogger(logging.LevelInfo)

	// Create context
	ctx := context.Background()

	// Create graph config
	graphConfig := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      getEnvOrDefault("NEO4J_URL", "bolt://localhost:7687"),
		Neo4jUser:     getEnvOrDefault("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnvOrDefault("NEO4J_PASSWORD", "testpass"),
	}

	// Create graph instance
	graphInstance, err := graph.NewNeo4jGraph(ctx, graphConfig, logger)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Neo4j: %v", err))
	}
	defer graphInstance.Close(ctx)

	session := graphInstance.Driver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	fmt.Println("=== NEO4J FULL DATABASE ANALYSIS ===")

	// Analyze all nodes with properties
	analyzeAllNodes(ctx, session)
	
	// Analyze all relationships
	analyzeAllRelationships(ctx, session)
	
	// Analyze conversation flow
	analyzeConversationFlow(ctx, session)
	
	// Analyze consistency issues  
	analyzeConsistencyIssues(ctx, session)
}

func analyzeAllNodes(ctx context.Context, session neo4j.SessionManagedTransaction) {
	query := `
		MATCH (n) 
		RETURN labels(n) as labels, properties(n) as props
		ORDER BY labels(n)[0], n.created_at
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		records, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}

		var nodes []map[string]interface{}
		for records.Next(ctx) {
			record := records.Record()
			nodes = append(nodes, map[string]interface{}{
				"labels": record.Values[0],
				"props":  record.Values[1],
			})
		}
		return nodes, records.Err()
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to query all nodes: %v", err))
	}

	nodes := result.([]map[string]interface{})

	fmt.Printf("\n📊 COMPLETE NODE ANALYSIS (%d total nodes):\n", len(nodes))
	for i, node := range nodes {
		labels := node["labels"]
		props := node["props"]
		
		// Pretty print the properties
		propsJSON, _ := json.MarshalIndent(props, "    ", "  ")
		fmt.Printf("\n  %d. Node Type: %v\n", i+1, labels)
		fmt.Printf("     Properties: %s\n", propsJSON)
	}
}

	t.Run("analyze_all_relationships_with_properties", func(t *testing.T) {
		query := `
			MATCH (a)-[r]->(b)
			RETURN 
				labels(a)[0] as from_type,
				a.id as from_id,
				COALESCE(a.name, a.title, a.id, 'no-name') as from_name,
				type(r) as rel_type,
				properties(r) as rel_props,
				labels(b)[0] as to_type,
				b.id as to_id,
				COALESCE(b.name, b.title, b.id, 'no-name') as to_name
			ORDER BY from_type, rel_type, to_type
		`

		result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			records, err := tx.Run(ctx, query, nil)
			if err != nil {
				return nil, err
			}

			var relationships []map[string]interface{}
			for records.Next(ctx) {
				record := records.Record()
				relationships = append(relationships, map[string]interface{}{
					"from_type": record.Values[0],
					"from_id":   record.Values[1],
					"from_name": record.Values[2],
					"rel_type":  record.Values[3],
					"rel_props": record.Values[4],
					"to_type":   record.Values[5],
					"to_id":     record.Values[6],
					"to_name":   record.Values[7],
				})
			}
			return relationships, records.Err()
		})

		require.NoError(t, err, "Failed to query all relationships")
		relationships := result.([]map[string]interface{})

		t.Logf("\n🔗 COMPLETE RELATIONSHIP ANALYSIS (%d total relationships):", len(relationships))
		for i, rel := range relationships {
			relPropsJSON, _ := json.MarshalIndent(rel["rel_props"], "      ", "  ")
			relPropsStr := string(relPropsJSON)
			t.Logf("\n  %d. %s(%s: %s) -[%s]-> %s(%s: %s)",
				i+1,
				rel["from_type"], rel["from_id"], rel["from_name"],
				rel["rel_type"],
				rel["to_type"], rel["to_id"], rel["to_name"])
			if relPropsStr != "null" && relPropsStr != "{}" {
				t.Logf("     Relationship Properties: %s", relPropsStr)
			}
		}
	})

	t.Run("analyze_conversation_flow", func(t *testing.T) {
		query := `
			MATCH (u:User)-[:PARTICIPANT_IN]->(c:Conversation)
			OPTIONAL MATCH (c)-[:CONTAINS_MESSAGE]->(m:ConversationMessage)
			OPTIONAL MATCH (c)-[:LINKED_TO_PLAN]->(p:execution_plan)
			OPTIONAL MATCH (p)<-[:CREATES_PLAN]-(d:Decision)
			RETURN 
				u.id as user_id,
				c.id as conversation_id,
				c.session_id as session_id,
				collect(DISTINCT m.id) as message_ids,
				collect(DISTINCT p.id) as plan_ids,
				collect(DISTINCT d.id) as decision_ids
		`

		result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			records, err := tx.Run(ctx, query, nil)
			if err != nil {
				return nil, err
			}

			var flows []map[string]interface{}
			for records.Next(ctx) {
				record := records.Record()
				flows = append(flows, map[string]interface{}{
					"user_id":        record.Values[0],
					"conversation_id": record.Values[1],
					"session_id":     record.Values[2],
					"message_ids":    record.Values[3],
					"plan_ids":       record.Values[4],
					"decision_ids":   record.Values[5],
				})
			}
			return flows, records.Err()
		})

		require.NoError(t, err, "Failed to query conversation flows")
		flows := result.([]map[string]interface{})

		t.Logf("\n💬 CONVERSATION FLOW ANALYSIS:")
		for i, flow := range flows {
			t.Logf("\n  %d. Conversation Flow:", i+1)
			t.Logf("     User: %s", flow["user_id"])
			t.Logf("     Conversation: %s (Session: %s)", flow["conversation_id"], flow["session_id"])
			t.Logf("     Messages: %v", flow["message_ids"])
			t.Logf("     Plans: %v", flow["plan_ids"])
			t.Logf("     Decisions: %v", flow["decision_ids"])
		}
	})

	t.Run("analyze_data_consistency_issues", func(t *testing.T) {
		// Check for nodes missing common naming fields
		query1 := `
			MATCH (n)
			WHERE n.id IS NOT NULL AND n.name IS NULL AND n.title IS NULL
			RETURN labels(n)[0] as node_type, count(*) as count
		`

		result1, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			records, err := tx.Run(ctx, query1, nil)
			if err != nil {
				return nil, err
			}

			var issues []map[string]interface{}
			for records.Next(ctx) {
				record := records.Record()
				issues = append(issues, map[string]interface{}{
					"node_type": record.Values[0],
					"count":     record.Values[1],
				})
			}
			return issues, records.Err()
		})

		require.NoError(t, err, "Failed to query naming issues")
		namingIssues := result1.([]map[string]interface{})

		t.Logf("\n⚠️  DATA CONSISTENCY ISSUES:")
		t.Logf("   Nodes missing 'name' or 'title' fields:")
		for _, issue := range namingIssues {
			t.Logf("     - %s: %v nodes", issue["node_type"], issue["count"])
		}

		// Check for redundant relationship data
		query2 := `
			MATCH (d:Decision)-[:CREATES_PLAN]->(p:execution_plan)
			WHERE d.execution_plan_id IS NOT NULL
			RETURN d.id as decision_id, d.execution_plan_id as prop_plan_id, p.id as edge_plan_id
		`

		result2, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			records, err := tx.Run(ctx, query2, nil)
			if err != nil {
				return nil, err
			}

			var redundancies []map[string]interface{}
			for records.Next(ctx) {
				record := records.Record()
				redundancies = append(redundancies, map[string]interface{}{
					"decision_id":   record.Values[0],
					"prop_plan_id":  record.Values[1],
					"edge_plan_id":  record.Values[2],
				})
			}
			return redundancies, records.Err()
		})

		require.NoError(t, err, "Failed to query redundancy issues")
		redundancies := result2.([]map[string]interface{})

		t.Logf("\n   Redundant relationship data (property + edge):")
		for _, redundancy := range redundancies {
			consistent := redundancy["prop_plan_id"] == redundancy["edge_plan_id"]
			status := "✅ CONSISTENT"
			if !consistent {
				status = "❌ INCONSISTENT"
			}
			t.Logf("     - Decision %s: property=%s, edge=%s %s",
				redundancy["decision_id"], redundancy["prop_plan_id"], redundancy["edge_plan_id"], status)
		}
	})
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
