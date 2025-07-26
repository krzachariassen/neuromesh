// UI-specific types for components
import { ExecutionPlanResponse } from './api';

export interface ChatMessage {
  id: string;
  content: string;
  role: 'user' | 'assistant' | 'system';
  timestamp: Date;
  metadata?: Record<string, any>;
}

export interface ChatState {
  messages: ChatMessage[];
  isLoading: boolean;
  error?: string;
  sessionId?: string;
  conversationId?: string;
}

export interface GraphViewState {
  selectedNodeId?: string;
  highlightedPath?: string[];
  filterBy?: 'all' | 'user' | 'conversation' | 'plan' | 'agent';
  isLoading: boolean;
  error?: string;
}

export interface OrchestrationState {
  currentExecution?: ExecutionPlanResponse;
  liveUpdates: boolean;
  agentStatuses: Record<string, AgentStatus>;
}

export interface AgentStatus {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'busy' | 'error';
  lastSeen: Date;
  currentTask?: string;
}

// WebSocket message types
export interface WebSocketMessage {
  type: 'agent_update' | 'execution_progress' | 'conversation_update';
  data: any;
  timestamp: string;
}

export interface ExecutionProgressUpdate {
  execution_id: string;
  step_id: string;
  status: 'started' | 'completed' | 'failed';
  progress?: number;
  message?: string;
}
