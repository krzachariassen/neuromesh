import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { GraphVisualization } from './GraphVisualization';
import '@testing-library/jest-dom';

// Mock API Service first
const mockGetGraphData = jest.fn();
const mockGetConversationHistory = jest.fn();
const mockGetExecutionPlans = jest.fn();

jest.mock('../../services/api', () => ({
  ApiService: jest.fn().mockImplementation(() => ({
    getGraphData: mockGetGraphData,
    getConversationHistory: mockGetConversationHistory,
    getExecutionPlans: mockGetExecutionPlans
  }))
}));

// Mock React Flow
jest.mock('reactflow', () => ({
  ReactFlow: ({ children, ...props }: any) => 
    <div data-testid="react-flow-wrapper" {...props}>{children}</div>,
  ReactFlowProvider: ({ children }: any) => 
    <div data-testid="react-flow-provider">{children}</div>,
  Controls: () => <div data-testid="react-flow-controls" />,
  MiniMap: () => <div data-testid="react-flow-minimap" />,
  Background: () => <div data-testid="react-flow-background" />,
  BackgroundVariant: { Dots: 'dots' },
  useNodesState: jest.fn(() => [[], jest.fn(), jest.fn()]),
  useEdgesState: jest.fn(() => [[], jest.fn(), jest.fn()]),
  addEdge: jest.fn(),
}));

describe('GraphVisualization Component - TDD RED PHASE', () => {
  beforeEach(() => {
    // Reset mocks before each test
    jest.clearAllMocks();
    
    // Set default mock values
    mockGetGraphData.mockResolvedValue({
      conversationId: 'test-conv-123',
      nodes: [
        {
          id: 'user-1',
          type: 'user',
          data: { name: 'Test User' },
          position: { x: 100, y: 100 }
        }
      ],
      edges: [
        {
          id: 'edge-1',
          source: 'user-1',
          target: 'conv-1',
          type: 'created'
        }
      ]
    });
    
    mockGetConversationHistory.mockResolvedValue([]);
    mockGetExecutionPlans.mockResolvedValue([]);
  });

  test('Should render React Flow wrapper', async () => {
    // GIVEN: GraphVisualization component
    render(<GraphVisualization conversationId="test-conv-123" />);

    // WHEN: Data loads
    await waitFor(() => {
      expect(screen.getByTestId('react-flow-wrapper')).toBeInTheDocument();
    });

    // THEN: Should render React Flow wrapper
    expect(screen.getByTestId('react-flow-wrapper')).toBeInTheDocument();
  });

  test('Should display loading state initially', () => {
    // GIVEN: GraphVisualization component
    render(<GraphVisualization conversationId="test-conv-123" />);

    // THEN: Should show loading indicator
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  test('Should fetch and display graph data', async () => {
    // GIVEN: Graph visualization component
    render(<GraphVisualization conversationId="test-conv-123" />);

    // WHEN: Component loads
    await waitFor(() => {
      expect(screen.getByTestId('react-flow-wrapper')).toBeInTheDocument();
    });

    // THEN: Should display the graph with nodes and edges
    expect(mockGetGraphData).toHaveBeenCalledWith('test-conv-123');
    expect(screen.getByTestId('react-flow-wrapper')).toBeInTheDocument();
  });

  test('Should display error state when graph data fails to load', async () => {
    // GIVEN: API call that fails
    mockGetGraphData.mockRejectedValue(new Error('Failed to load graph data'));

    // WHEN: Component renders
    render(<GraphVisualization conversationId="test-conv-123" />);

    // Debug: Let's check if our mock is being called
    await waitFor(() => {
      expect(mockGetGraphData).toHaveBeenCalled();
    }, { timeout: 1000 });

    // THEN: Should display error message
    await waitFor(() => {
      expect(screen.getByTestId('error')).toBeInTheDocument();
    }, { timeout: 3000 });
    
    expect(screen.getByText(/Failed to load graph data/i)).toBeInTheDocument();
    expect(screen.getByText(/Retry/i)).toBeInTheDocument();
  });

  test('Should render graph controls (zoom, pan, minimap)', async () => {
    // GIVEN: GraphVisualization component
    render(<GraphVisualization conversationId="test-conv-123" />);

    // WHEN: Component loads
    await waitFor(() => {
      expect(screen.getByTestId('react-flow-wrapper')).toBeInTheDocument();
    });

    // THEN: Should have all graph controls
    expect(screen.getByTestId('react-flow-controls')).toBeInTheDocument();
    expect(screen.getByTestId('react-flow-background')).toBeInTheDocument();
    expect(screen.getByTestId('react-flow-minimap')).toBeInTheDocument();
  });

  test('Should handle empty graph data gracefully', async () => {
    // GIVEN: Empty graph data response
    const emptyGraphData = { conversationId: 'test-conv-123', nodes: [], edges: [] };
    mockGetGraphData.mockResolvedValue(emptyGraphData);

    // WHEN: Component renders
    render(<GraphVisualization conversationId="test-conv-123" />);

    // THEN: Should handle empty data gracefully
    await waitFor(() => {
      expect(screen.getByTestId('react-flow-wrapper')).toBeInTheDocument();
    }, { timeout: 3000 });
    
    // React Flow should still render but with no nodes/edges
    expect(screen.getByTestId('react-flow-wrapper')).toBeInTheDocument();
  });
});
