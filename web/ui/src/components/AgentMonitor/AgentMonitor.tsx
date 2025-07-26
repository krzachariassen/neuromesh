import { useState, useEffect } from 'react';
import { CheckCircleIcon, XCircleIcon, ClockIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline';

interface Agent {
  name: string;
  type: string;
  status: string;
  capabilities: string[];
  metadata: {
    last_active: string;
  };
}

interface AgentResponse {
  agents: Agent[];
}

const AgentMonitor: React.FC = () => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchAgentStatus = async () => {
    try {
      console.log('AgentMonitor: Fetching agent status from /api/agents/status');
      setIsLoading(true);
      setError(null);
      
      const response = await fetch('/api/agents/status');
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      const data: AgentResponse = await response.json();
      console.log('AgentMonitor: Received agent data:', data);
      
      setAgents(data.agents || []);
    } catch (err) {
      console.error('AgentMonitor: Error fetching agent status:', err);
      setError('Error loading agents. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchAgentStatus();
  }, []);

  const handleRefresh = () => {
    fetchAgentStatus();
  };

  // Calculate counts from real data
  const onlineCount = agents.filter(agent => agent.status === 'active').length;
  const busyCount = agents.filter(agent => agent.status === 'busy').length;
  const offlineCount = agents.filter(agent => 
    agent.status !== 'active' && agent.status !== 'busy'
  ).length;

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <CheckCircleIcon className="h-5 w-5 text-green-500" />;
      case 'busy':
        return <ClockIcon className="h-5 w-5 text-yellow-500" />;
      case 'idle':
        return <XCircleIcon className="h-5 w-5 text-gray-400" />;
      case 'error':
        return <ExclamationTriangleIcon className="h-5 w-5 text-red-500" />;
      default:
        return <XCircleIcon className="h-5 w-5 text-gray-400" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'text-green-500';
      case 'busy':
        return 'text-yellow-500';
      case 'idle':
        return 'text-gray-400';
      case 'error':
        return 'text-red-500';
      default:
        return 'text-gray-400';
    }
  };

  const formatLastSeen = (lastActive: string) => {
    try {
      const date = new Date(lastActive);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / 60000);
      
      if (diffMins < 1) return 'Just now';
      if (diffMins < 60) return `${diffMins} min ago`;
      
      const diffHours = Math.floor(diffMins / 60);
      if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
      
      const diffDays = Math.floor(diffHours / 24);
      return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
    } catch {
      return 'Unknown';
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold text-gray-900">Agent Monitor</h2>
          <button 
            onClick={handleRefresh}
            className="text-sm text-blue-600 hover:text-blue-800"
          >
            Refresh Status
          </button>
        </div>
        <div className="text-center py-8">
          <p className="text-gray-500">Loading agent status...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold text-gray-900">Agent Monitor</h2>
          <button 
            onClick={handleRefresh}
            className="text-sm text-blue-600 hover:text-blue-800"
          >
            Refresh Status
          </button>
        </div>
        <div className="text-center py-8">
          <p className="text-red-500">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-gray-900">Agent Monitor</h2>
        <button 
          onClick={handleRefresh}
          className="text-sm text-blue-600 hover:text-blue-800"
        >
          Refresh Status
        </button>
      </div>

      {/* Agent Status Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-green-50 p-4 rounded-lg">
          <div className="flex items-center">
            <CheckCircleIcon className="h-8 w-8 text-green-500" />
            <div className="ml-3">
              <p className="text-sm font-medium text-green-900">Online</p>
              <p className="text-2xl font-bold text-green-900">{onlineCount}</p>
            </div>
          </div>
        </div>
        
        <div className="bg-yellow-50 p-4 rounded-lg">
          <div className="flex items-center">
            <ClockIcon className="h-8 w-8 text-yellow-500" />
            <div className="ml-3">
              <p className="text-sm font-medium text-yellow-900">Busy</p>
              <p className="text-2xl font-bold text-yellow-900">{busyCount}</p>
            </div>
          </div>
        </div>
        
        <div className="bg-gray-50 p-4 rounded-lg">
          <div className="flex items-center">
            <XCircleIcon className="h-8 w-8 text-gray-400" />
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-900">Offline</p>
              <p className="text-2xl font-bold text-gray-900">{offlineCount}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Agent List */}
      <div className="bg-white shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
            Active Agents ({agents.length})
          </h3>
          
          {agents.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-gray-500">No agents currently registered</p>
            </div>
          ) : (
            <div className="space-y-4">
              {agents.map((agent, index) => (
                <div
                  key={`${agent.name}-${index}`}
                  className="flex items-center justify-between p-4 border border-gray-200 rounded-lg"
                >
                  <div className="flex items-center space-x-3">
                    {getStatusIcon(agent.status)}
                    <div>
                      <h4 className="text-sm font-medium text-gray-900">{agent.name}</h4>
                      <p className="text-sm text-gray-500">Type: {agent.type}</p>
                      {agent.capabilities && agent.capabilities.length > 0 && (
                        <p className="text-xs text-gray-400">
                          Capabilities: {agent.capabilities.join(', ')}
                        </p>
                      )}
                    </div>
                  </div>
                  
                  <div className="text-right">
                    <span className={`text-sm font-medium ${getStatusColor(agent.status)}`}>
                      {agent.status.charAt(0).toUpperCase() + agent.status.slice(1)}
                    </span>
                    <p className="text-xs text-gray-500">
                      Last seen: {formatLastSeen(agent.metadata.last_active)}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AgentMonitor;
