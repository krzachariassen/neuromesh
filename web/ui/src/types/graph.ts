// Graph visualization types for React Flow integration
export interface GraphNode {
  id: string;
  type: 'user' | 'conversation' | 'execution_plan' | 'execution_step' | 'agent' | 'result';
  data: {
    name?: string;
    title?: string;
    status?: string;
    [key: string]: any;
  };
  position: { x: number; y: number };
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: 'created' | 'linked_to' | 'executed' | 'synthesized';
  data?: {
    [key: string]: any;
  };
}

export interface GraphDataResponse {
  conversationId: string;
  nodes: GraphNode[];
  edges: GraphEdge[];
}

// React Flow specific types
export interface ReactFlowNode extends GraphNode {
  // React Flow will add additional properties
}
