import React, { useState, useEffect } from 'react';

interface Agent {
  name: string;
  type: string;
  status: string;
  capabilities: string[];
  metadata: {
    last_active: string;
  };
}

interface AgentStatusResponse {
  agents: Agent[];
}

interface HealthResponse {
  status: string;
  service: string;
}

const Dashboard: React.FC = () => {
  console.log('🎯 Dashboard: Component mounted/re-rendered');
  const [agentCount, setAgentCount] = useState<number | null>(null);
  const [isLoadingAgents, setIsLoadingAgents] = useState(true);
  const [agentError, setAgentError] = useState<string | null>(null);
  const [systemHealth, setSystemHealth] = useState<HealthResponse | null>(null);

  const fetchAgentStatus = async () => {
    console.log('🚀 Dashboard: Starting fetchAgentStatus...');
    try {
      setIsLoadingAgents(true);
      setAgentError(null);
      console.log('📡 Dashboard: Making fetch request to /api/agents/status');
      const response = await fetch('/api/agents/status');
      console.log('📦 Dashboard: Received response:', response.status, response.statusText);
      if (response.ok) {
        const data: AgentStatusResponse = await response.json();
        console.log('✅ Dashboard: Successfully fetched agent data:', data);
        setAgentCount(data.agents.length);
      } else {
        const errorMsg = `API Error: ${response.status}`;
        console.error('❌ Dashboard: API error:', errorMsg);
        setAgentError(errorMsg);
      }
    } catch (error) {
      console.error('💥 Dashboard: Fetch failed:', error);
      setAgentError('Connection failed - API server offline');
    } finally {
      setIsLoadingAgents(false);
      console.log('🏁 Dashboard: fetchAgentStatus completed');
    }
  };

  useEffect(() => {
    console.log('🔄 Dashboard: useEffect triggered, starting data fetch...');
    // Fetch system health
    const fetchSystemHealth = async () => {
      try {
        console.log('🏥 Dashboard: Fetching system health...');
        const response = await fetch('/health');
        if (response.ok) {
          const data: HealthResponse = await response.json();
          console.log('✅ Dashboard: Health check successful:', data);
          setSystemHealth(data);
        }
      } catch (error) {
        console.error('❌ Dashboard: Health check failed:', error);
        // Keep default state
      }
    };

    fetchAgentStatus();
    fetchSystemHealth();
  }, []);

  return (
    <div data-testid="dashboard-view" className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
        <div className="flex items-center space-x-4">
          <div className="text-sm text-gray-500">
            NeuroMesh AI Orchestration Platform
          </div>
          {agentError && (
            <button 
              onClick={fetchAgentStatus}
              className="text-sm bg-blue-600 text-white px-3 py-1 rounded hover:bg-blue-700"
              disabled={isLoadingAgents}
            >
              {isLoadingAgents ? 'Retrying...' : 'Retry Connection'}
            </button>
          )}
        </div>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="card">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-500">Active Agents</p>
              {isLoadingAgents ? (
                <p className="text-2xl font-bold text-gray-400">Loading...</p>
              ) : agentError ? (
                <div>
                  <p className="text-2xl font-bold text-red-500">—</p>
                  <p className="text-xs text-red-500">{agentError}</p>
                </div>
              ) : (
                <p className="text-2xl font-bold text-gray-900">{agentCount}</p>
              )}
            </div>
            <div className={`h-8 w-8 rounded-full flex items-center justify-center ${
              isLoadingAgents ? 'bg-gray-100' : agentError ? 'bg-red-100' : 'bg-green-100'
            }`}>
              <div className={`h-3 w-3 rounded-full ${
                isLoadingAgents ? 'bg-gray-400' : agentError ? 'bg-red-500' : 'bg-green-500'
              }`}></div>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-500">Conversations</p>
              <p className="text-2xl font-bold text-gray-900">12</p>
            </div>
            <div className="h-8 w-8 bg-blue-100 rounded-full flex items-center justify-center">
              <div className="h-3 w-3 bg-blue-500 rounded-full"></div>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-500">Executions</p>
              <p className="text-2xl font-bold text-gray-900">8</p>
            </div>
            <div className="h-8 w-8 bg-yellow-100 rounded-full flex items-center justify-center">
              <div className="h-3 w-3 bg-yellow-500 rounded-full"></div>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-500">Success Rate</p>
              <p className="text-2xl font-bold text-gray-900">94%</p>
            </div>
            <div className="h-8 w-8 bg-purple-100 rounded-full flex items-center justify-center">
              <div className="h-3 w-3 bg-purple-500 rounded-full"></div>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Recent Conversations</h3>
          <div className="space-y-3">
            {[
              { id: 'conv-1', preview: 'Healthcare diagnosis request...', time: '2 min ago' },
              { id: 'conv-2', preview: 'Data analysis workflow...', time: '5 min ago' },
              { id: 'conv-3', preview: 'System deployment planning...', time: '10 min ago' },
            ].map((conv) => (
              <div key={conv.id} className="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
                <div>
                  <p className="text-sm font-medium text-gray-900">{conv.preview}</p>
                  <p className="text-xs text-gray-500">{conv.time}</p>
                </div>
                <button className="text-primary-600 hover:text-primary-700 text-sm font-medium">
                  View
                </button>
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <h3 className="text-lg font-medium text-gray-900 mb-4">System Health</h3>
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-sm text-gray-600">AI Provider</span>
              <span className="text-sm font-medium text-green-600">Healthy</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-sm text-gray-600">Graph Database</span>
              <span className="text-sm font-medium text-green-600">Healthy</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-sm text-gray-600">Message Bus</span>
              <span className="text-sm font-medium text-green-600">Healthy</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-sm text-gray-600">Agents Registry</span>
              <span className="text-sm font-medium text-green-600">Healthy</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
