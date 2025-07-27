import React, { useCallback, useEffect, useState, useMemo } from 'react';
import {
  ReactFlow,
  Node,
  Edge,
  addEdge,
  Connection,
  useNodesState,
  useEdgesState,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  ReactFlowProvider,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { ApiService } from '../../services/api';
import { GraphDataResponse, GraphNode, GraphEdge } from '../../types/graph';

interface GraphVisualizationProps {
  conversationId: string;
}

const nodeTypes = {
  user: { color: '#1f77b4' },
  conversation: { color: '#ff7f0e' },
  execution_plan: { color: '#2ca02c' },
  execution_step: { color: '#d62728' },
  agent: { color: '#9467bd' },
  result: { color: '#8c564b' },
};

export const GraphVisualization: React.FC<GraphVisualizationProps> = ({
  conversationId
}) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create ApiService once using useMemo to prevent infinite re-renders
  const apiService = useMemo(() => new ApiService(), []);

  const loadGraphData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const graphData = await apiService.getGraphData(conversationId);
      
      // Transform nodes for React Flow directly in the function
      const reactFlowNodes: Node[] = graphData.nodes.map((node: GraphNode) => ({
        id: node.id,
        type: 'default',
        data: {
          label: node.data.name || node.data.title || node.id,
          ...node.data
        },
        position: node.position,
        style: {
          background: nodeTypes[node.type]?.color || '#f0f0f0',
          color: 'white',
          border: '1px solid #222138',
          borderRadius: '3px',
          fontSize: '12px',
          padding: '10px',
        },
      }));

      // Transform edges for React Flow directly in the function
      const reactFlowEdges: Edge[] = graphData.edges.map((edge: GraphEdge) => ({
        id: edge.id,
        source: edge.source,
        target: edge.target,
        type: 'default',
        animated: edge.type === 'executed',
        style: {
          stroke: edge.type === 'created' ? '#1f77b4' : '#666',
          strokeWidth: 2,
        },
        label: edge.type,
      }));
      
      setNodes(reactFlowNodes);
      setEdges(reactFlowEdges);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load graph data');
    } finally {
      setLoading(false);
    }
  }, [conversationId, apiService]);

  useEffect(() => {
    if (conversationId) {
      loadGraphData();
    }
  }, [conversationId, loadGraphData]);

  const onConnect = useCallback(
    (params: Edge | Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  if (loading) {
    return (
      <div className="graph-visualization-loading" data-testid="loading">
        Loading graph data...
      </div>
    );
  }

  if (error) {
    return (
      <div className="graph-visualization-error" data-testid="error">
        Error: {error}
        <button onClick={loadGraphData}>Retry</button>
      </div>
    );
  }

  return (
    <div className="graph-visualization" style={{ width: '100%', height: '600px' }}>
      <ReactFlowProvider>
        <ReactFlow
          data-testid="react-flow-wrapper"
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          fitView
        >
          <Controls />
          <MiniMap />
          <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
};
