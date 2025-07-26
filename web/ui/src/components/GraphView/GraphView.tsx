import React from 'react';

const GraphView: React.FC = () => {
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

      {/* Graph Container */}
      <div className="card h-96">
        <div className="h-full flex items-center justify-center bg-gray-50 rounded-lg border-2 border-dashed border-gray-300">
          <div className="text-center">
            <div className="text-gray-400 mb-2">
              <svg className="mx-auto h-12 w-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 20l-5.447-2.724A1 1 0 013 16.382V7.618a1 1 0 01.553-.894L9 4l6 3 6-3 .553.894A1 1 0 0122 7.618v8.764a1 1 0 01-.553.894L15 20l-6-3z" />
              </svg>
            </div>
            <p className="text-sm text-gray-500">Graph visualization will be rendered here</p>
            <p className="text-xs text-gray-400 mt-1">Connect to view conversation flows and agent interactions</p>
          </div>
        </div>
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
