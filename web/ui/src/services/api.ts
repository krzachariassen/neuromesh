import { GraphDataResponse } from '../types/graph';

const API_BASE = 'http://localhost:8080';

export class ApiService {
  async getGraphData(conversationId: string): Promise<GraphDataResponse> {
    try {
      // Use the clean REST API endpoint instead of the old /api/ui/ pattern
      const response = await fetch(`${API_BASE}/api/v1/conversations/${conversationId}/graph`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error('Error fetching graph data:', error);
      throw error;
    }
  }

  async getConversationHistory(conversationId: string): Promise<any[]> {
    // Use clean REST API endpoint - conversation details should contain history
    const response = await fetch(`${API_BASE}/api/v1/conversations/${conversationId}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch conversation: ${response.statusText}`);
    }
    const conversation = await response.json();
    return conversation.history || [];
  }

  async getExecutionPlans(conversationId: string): Promise<any[]> {
    // Use clean REST API endpoint for execution plans
    const response = await fetch(`${API_BASE}/api/v1/conversations/${conversationId}/execution-plans`);
    if (!response.ok) {
      throw new Error(`Failed to fetch execution plans: ${response.statusText}`);
    }
    return response.json();
  }
}
