package application

import (
	"context"
	"testing"

	"neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"

	"github.com/stretchr/testify/assert"
)

func TestAIDecisionEngine_MultiAgentOrchestration(t *testing.T) {
	t.Run("should create multi-step execution plan for complex user request using real AI", func(t *testing.T) {
		// Arrange: Set up real AI and mock repository
		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		engine := NewAIDecisionEngineWithRepository(aiProvider, mockRepo)

		// Complex user request requiring multiple agents
		userInput := `Count the words in this text 'Hi you' then translate it into Danish and send to my email ka@ka.dk`
		userID := "user123"
		requestID := "multi-agent-request-123"

		// Agent context with the agents we expect to be used
		agentContext := `Available Agents:
- text-analysis-agent | Status: available | Capabilities: word count, text analysis, content processing
- translation-agent | Status: available | Capabilities: translate text, language conversion, multi-language support
- email-agent | Status: available | Capabilities: send email, email delivery, notification services
- deploy-agent | Status: available | Capabilities: deployment, server management`

		// Act: Analyze the request
		analysis, err := engine.ExploreAndAnalyze(ctx, userInput, userID, agentContext, requestID)
		assert.NoError(t, err)
		assert.NotNil(t, analysis)

		t.Logf("AI Analysis:")
		t.Logf("  Intent: %s", analysis.Intent)
		t.Logf("  Category: %s", analysis.Category)
		t.Logf("  Confidence: %d", analysis.Confidence)
		t.Logf("  Required Agents: %v", analysis.RequiredAgents)
		t.Logf("  Reasoning: %s", analysis.Reasoning)

		// Act: Make decision and create execution plan
		decision, err := engine.MakeDecision(ctx, userInput, userID, analysis, requestID)
		assert.NoError(t, err)
		assert.NotNil(t, decision)

		// Debug: Log the AI's execution plan text before parsing
		t.Logf("\nAI Decision Details:")
		t.Logf("  Type: %s", decision.Type)
		t.Logf("  Reasoning: %s", decision.Reasoning)
		t.Logf("  Agent Coordination: %s", decision.AgentCoordination)

		// Assert: Should be an execute decision
		assert.Equal(t, domain.DecisionTypeExecute, decision.Type)

		// Assert: ExecutionPlan should be created and persisted
		assert.Len(t, decision.ExecutionPlanID, 36, "ExecutionPlanID should be a UUID")
		assert.NotContains(t, decision.ExecutionPlanID, "Step", "ExecutionPlanID should not contain execution plan text")

		// Assert: Repository interactions
		assert.Equal(t, 1, mockRepo.GetPlanCount(), "One ExecutionPlan should have been created")
		assert.Equal(t, 1, mockRepo.GetLinkCount(), "ExecutionPlan should be linked to analysis")

		// Retrieve the created execution plan to examine its structure
		retrievedPlan, err := mockRepo.GetByID(ctx, decision.ExecutionPlanID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPlan)

		t.Logf("\nCreated ExecutionPlan:")
		t.Logf("  ID: %s", retrievedPlan.ID)
		t.Logf("  Name: %s", retrievedPlan.Name)
		t.Logf("  Description: %s", retrievedPlan.Description)
		t.Logf("  Status: %s", retrievedPlan.Status)
		t.Logf("  Priority: %s", retrievedPlan.Priority)
		t.Logf("  Number of Steps: %d", len(retrievedPlan.Steps))

		// Assert: Plan should have multiple steps
		assert.GreaterOrEqual(t, len(retrievedPlan.Steps), 2, "Should have multiple steps for complex workflow")

		// Log each step to see the AI's planning
		for i, step := range retrievedPlan.Steps {
			t.Logf("\n  Step %d:", i+1)
			t.Logf("    ID: %s", step.ID)
			t.Logf("    Name: %s", step.Name)
			t.Logf("    Description: %s", step.Description)
			t.Logf("    Assigned Agent: %s", step.AssignedAgent)
			t.Logf("    Status: %s", step.Status)
			t.Logf("    Step Number: %d", step.StepNumber)
		}

		// Assert: Steps should be assigned to appropriate agents
		// We can't predict exact agent assignments since we're using real AI,
		// but we can validate the structure
		for _, step := range retrievedPlan.Steps {
			assert.NotEmpty(t, step.Name, "Step should have a name")
			assert.NotEmpty(t, step.Description, "Step should have a description")
			assert.Greater(t, step.StepNumber, 0, "Step should have a valid step number")
			assert.Equal(t, domain.ExecutionStepStatusPending, step.Status, "New steps should be pending")
		}

		t.Logf("\n✅ Multi-agent orchestration test completed successfully!")
		t.Logf("   The AI successfully created a structured execution plan with %d steps", len(retrievedPlan.Steps))
		t.Logf("   Each step is properly structured and ready for agent execution")
	})

	t.Run("should handle complex CI/CD pipeline with multiple agents and dependencies", func(t *testing.T) {
		// Arrange: Set up real AI and mock repository
		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		engine := NewAIDecisionEngineWithRepository(aiProvider, mockRepo)

		// Complex CI/CD request requiring multiple specialized agents
		userInput := `Deploy my web application to production with full CI/CD pipeline: run security scan on code, execute unit tests, build Docker image, deploy to staging environment, run integration tests, perform load testing, scan for vulnerabilities in deployment, deploy to production, configure monitoring and alerts, and send deployment notification to team Slack channel`
		userID := "devops-user-456"
		requestID := "cicd-pipeline-request-789"

		// Comprehensive agent context with specialized CI/CD agents
		agentContext := `Available Agents:
- security-scan-agent | Status: available | Capabilities: static code analysis, vulnerability scanning, security auditing, SAST scanning
- test-runner-agent | Status: available | Capabilities: unit testing, integration testing, test execution, test reporting
- build-agent | Status: available | Capabilities: Docker build, container creation, artifact building, compilation
- deployment-agent | Status: available | Capabilities: application deployment, environment management, infrastructure provisioning
- load-test-agent | Status: available | Capabilities: performance testing, load testing, stress testing, capacity planning
- vulnerability-scanner-agent | Status: available | Capabilities: runtime vulnerability scanning, container scanning, dependency analysis
- monitoring-agent | Status: available | Capabilities: metrics setup, alerting configuration, observability, health checks
- notification-agent | Status: available | Capabilities: Slack notifications, email alerts, team communication, status reporting
- database-agent | Status: available | Capabilities: database migration, schema updates, data management
- backup-agent | Status: available | Capabilities: data backup, disaster recovery, snapshot management`

		// Act: Analyze the complex request
		analysis, err := engine.ExploreAndAnalyze(ctx, userInput, userID, agentContext, requestID)
		assert.NoError(t, err)
		assert.NotNil(t, analysis)

		t.Logf("\n🔍 AI Analysis for Complex CI/CD Pipeline:")
		t.Logf("  Intent: %s", analysis.Intent)
		t.Logf("  Category: %s", analysis.Category)
		t.Logf("  Confidence: %d%%", analysis.Confidence)
		t.Logf("  Required Agents: %v", analysis.RequiredAgents)
		t.Logf("  Reasoning: %s", analysis.Reasoning)

		// Assert: Should identify multiple agents for complex workflow
		assert.GreaterOrEqual(t, len(analysis.RequiredAgents), 5, "Complex CI/CD should require multiple specialized agents")
		assert.GreaterOrEqual(t, analysis.Confidence, 80, "AI should be confident about CI/CD workflow")

		// Act: Make decision and create execution plan
		decision, err := engine.MakeDecision(ctx, userInput, userID, analysis, requestID)
		assert.NoError(t, err)
		assert.NotNil(t, decision)

		t.Logf("\n🎯 AI Decision for CI/CD Pipeline:")
		t.Logf("  Type: %s", decision.Type)
		t.Logf("  Reasoning: %s", decision.Reasoning)

		// Assert: Should be an execute decision
		assert.Equal(t, domain.DecisionTypeExecute, decision.Type)

		// Assert: ExecutionPlan should be created and persisted
		assert.Len(t, decision.ExecutionPlanID, 36, "ExecutionPlanID should be a UUID")
		assert.NotContains(t, decision.ExecutionPlanID, "Step", "ExecutionPlanID should not contain execution plan text")

		// Assert: Repository interactions
		assert.Equal(t, 1, mockRepo.GetPlanCount(), "One ExecutionPlan should have been created")
		assert.Equal(t, 1, mockRepo.GetLinkCount(), "ExecutionPlan should be linked to analysis")

		// Retrieve the created execution plan to examine its complex structure
		retrievedPlan, err := mockRepo.GetByID(ctx, decision.ExecutionPlanID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPlan)

		t.Logf("\n📋 Created Complex ExecutionPlan:")
		t.Logf("  ID: %s", retrievedPlan.ID)
		t.Logf("  Name: %s", retrievedPlan.Name)
		t.Logf("  Description: %s", retrievedPlan.Description)
		t.Logf("  Status: %s", retrievedPlan.Status)
		t.Logf("  Priority: %s", retrievedPlan.Priority)
		t.Logf("  Number of Steps: %d", len(retrievedPlan.Steps))

		// Assert: Complex pipeline should have many steps
		assert.GreaterOrEqual(t, len(retrievedPlan.Steps), 6, "Complex CI/CD pipeline should have multiple steps")
		assert.LessOrEqual(t, len(retrievedPlan.Steps), 15, "Should be reasonable number of steps")

		// Track unique agents used across all steps
		uniqueAgents := make(map[string]bool)
		stepsByAgent := make(map[string][]string)

		// Log each step to see the AI's complex planning
		t.Logf("\n🔧 Detailed CI/CD Pipeline Steps:")
		for i, step := range retrievedPlan.Steps {
			t.Logf("\n  Step %d:", i+1)
			t.Logf("    ID: %s", step.ID)
			t.Logf("    Name: %s", step.Name)
			t.Logf("    Description: %s", step.Description)
			t.Logf("    Assigned Agent: %s", step.AssignedAgent)
			t.Logf("    Status: %s", step.Status)
			t.Logf("    Step Number: %d", step.StepNumber)

			// Track agent usage
			uniqueAgents[step.AssignedAgent] = true
			stepsByAgent[step.AssignedAgent] = append(stepsByAgent[step.AssignedAgent], step.Name)
		}

		// Assert: Complex workflow should use multiple different agents
		assert.GreaterOrEqual(t, len(uniqueAgents), 4, "Complex CI/CD should utilize multiple different agents")

		t.Logf("\n📊 Agent Utilization Summary:")
		t.Logf("  Total Unique Agents: %d", len(uniqueAgents))
		for agent, steps := range stepsByAgent {
			t.Logf("  - %s: %d steps (%v)", agent, len(steps), steps)
		}

		// Assert: All steps should be properly structured
		for i, step := range retrievedPlan.Steps {
			assert.NotEmpty(t, step.Name, "Step %d should have a name", i+1)
			assert.NotEmpty(t, step.Description, "Step %d should have a description", i+1)
			assert.NotEmpty(t, step.AssignedAgent, "Step %d should have an assigned agent", i+1)
			assert.Greater(t, step.StepNumber, 0, "Step %d should have a valid step number", i+1)
			assert.Equal(t, domain.ExecutionStepStatusPending, step.Status, "Step %d should be pending", i+1)

			// Validate agent names are properly extracted (should contain '-agent' suffix)
			assert.Contains(t, step.AssignedAgent, "-agent", "Agent name should follow proper naming convention")
			assert.NotContains(t, step.AssignedAgent, " ", "Agent name should not contain spaces")
		}

		// Log agent coordination details
		t.Logf("\n🤝 Agent Coordination Strategy:")
		t.Logf("  %s", decision.AgentCoordination)

		t.Logf("\n✅ Complex CI/CD multi-agent orchestration test completed successfully!")
		t.Logf("   The AI successfully orchestrated a %d-step pipeline using %d different agents", len(retrievedPlan.Steps), len(uniqueAgents))
		t.Logf("   Each step is properly structured with appropriate agent assignments")
		t.Logf("   The system demonstrates capability to handle enterprise-grade workflows")
	})

	// NOTE: Healthcare demonstration has been moved to healthcare_demo_test.go
	// This keeps the multi-agent orchestration tests focused on the core coordination mechanics
	// while the healthcare demo showcases our platform's vision for specialized agent workflows
}
