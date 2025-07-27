import React from 'react';
import { GraphVisualization } from './GraphVisualization';

const GraphView: React.FC = () => {
  // For now, we'll use a default conversation ID
  // TODO: This should come from URL params or user selection
  const conversationId = "test-conversation-1";

  return (
    <div data-testid="graph-view" className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold text-gray-900">Graph View</h1>
        <div className="flex space-x-2">
          <button className="btn-secondary">
            Reset View
          </button>
          <button className="btn-primary">
            Export
          </button>
        </div>
      </div>

      {/* Graph Container - Now using our React Flow component */}
      <div className="card" style={{ height: '600px' }}>
        <GraphVisualization conversationId={conversationId} />
      </div>

      {/* Graph Controls */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="card">
          <h3 className="text-lg font-medium text-gray-900 mb-4">View Options</h3>
          <div className="space-y-3">
            <label className="flex items-center">
              <input type="checkbox" className="rounded border-gray-300 text-primary-600 focus:ring-primary-500" defaultChecked />
              <span className="ml-2 text-sm text-gray-700">Show Agents</span>
            </label>
            <label className="flex items-center">
              <input type="checkbox" className="rounded border-gray-300 text-primary-600 focus:ring-primary-500" defaultChecked />
              <span className="ml-2 text-sm text-gray-700">Show Conversations</span>
            </label>
            <label className="flex items-center">
              <input type="checkbox" className="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              <span className="ml-2 text-sm text-gray-700">Show Execution Plans</span>
            </label>
          </div>
        </div>

        <div className="card">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Filters</h3>
          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Time Range</label>
              <select className="w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500">
                <option>Last Hour</option>
                <option>Last 24 Hours</option>
                <option>Last Week</option>
                <option>All Time</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Agent Type</label>
              <select className="w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500">
                <option>All Agents</option>
                <option>Text Processors</option>
                <option>Data Analyzers</option>
                <option>Decision Makers</option>
              </select>
            </div>
          </div>
        </div>

        <div className="card">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Graph Stats</h3>
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">Nodes</span>
              <span className="text-sm font-medium">24</span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">Edges</span>
              <span className="text-sm font-medium">18</span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">Connected Components</span>
              <span className="text-sm font-medium">3</span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">Max Depth</span>
              <span className="text-sm font-medium">5</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default GraphView;
