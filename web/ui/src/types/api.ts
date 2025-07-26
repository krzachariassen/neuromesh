// API Types - matching our Go backend DTOs

export interface GraphDataResponse {
  conversation_id: string;
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface GraphNode {
  id: string;
  type: 'user' | 'conversation' | 'plan' | 'step' | 'agent' | 'result';
  data: Record<string, any>;
  position?: NodePosition;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: 'created' | 'executed' | 'synthesized';
  data?: Record<string, any>;
}

export interface NodePosition {
  x: number;
  y: number;
}

export interface ExecutionPlanResponse {
  id: string;
  name: string;
  description?: string;
  status: 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED';
  created_at: string;
  steps: ExecutionStepData[];
}

export interface ExecutionStepData {
  step_number: number;
  name: string;
  description: string;
  agent_name: string;
  status: 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED';
  completed_at?: string;
}

export interface ConversationHistoryResponse {
  session_id: string;
  conversations: ConversationData[];
  messages: MessageData[];
}

export interface ConversationData {
  id: string;
  session_id: string;
  user_id: string;
  status: 'active' | 'completed' | 'archived';
  created_at: string;
}

export interface MessageData {
  id: string;
  conversation_id: string;
  role: 'user' | 'assistant';
  content: string;
  metadata: Record<string, any>;
  created_at: string;
}

export interface AgentStatusResponse {
  agents: AgentData[];
}

export interface AgentData {
  name: string;
  type: string;
  status: 'active' | 'inactive' | 'error';
  capabilities: string[];
  metadata: Record<string, any>;
}
