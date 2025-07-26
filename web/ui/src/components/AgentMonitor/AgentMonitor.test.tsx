import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import AgentMonitor from './AgentMonitor';

// Mock fetch globally for all tests
global.fetch = jest.fn();
const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

describe('AgentMonitor Component', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.log = jest.fn(); // Mock console.log for cleaner test output
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('should display loading state initially', () => {
    // Mock fetch to never resolve (simulating loading)
    mockFetch.mockImplementation(() => new Promise(() => {}));

    render(<AgentMonitor />);
    
    expect(screen.getByText('Loading agent status...')).toBeInTheDocument();
  });

  it('should fetch and display real agent data from API', async () => {
    // RED: This test will fail because current component uses hardcoded data
    const mockAgents = [
      {
        name: 'Real Text Processor',
        type: 'text-analysis',
        status: 'active',
        capabilities: ['text-processing', 'sentiment-analysis'],
        metadata: {
          last_active: '2024-01-15T10:30:00Z'
        }
      },
      {
        name: 'Real Data Analyzer', 
        type: 'data-analysis',
        status: 'idle',
        capabilities: ['data-mining', 'pattern-recognition'],
        metadata: {
          last_active: '2024-01-15T10:25:00Z'
        }
      }
    ];

    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ agents: mockAgents }),
    } as Response);

    render(<AgentMonitor />);

    // Wait for the API call to complete
    await waitFor(() => {
      expect(screen.getByText('Real Text Processor')).toBeInTheDocument();
    });

    // Verify real API data is displayed, not hardcoded mock data
    expect(screen.getByText('Real Text Processor')).toBeInTheDocument();
    expect(screen.getByText('Real Data Analyzer')).toBeInTheDocument();
    
    // Verify the API was called with correct endpoint
    expect(mockFetch).toHaveBeenCalledWith('/api/agents/status');
  });

  it('should handle API errors gracefully', async () => {
    // Mock API failure
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    render(<AgentMonitor />);

    await waitFor(() => {
      expect(screen.getByText(/error loading agents/i)).toBeInTheDocument();
    });
  });

  it('should show correct status counts from real data', async () => {
    const mockAgents = [
      {
        name: 'Agent 1',
        type: 'test',
        status: 'active',
        capabilities: [],
        metadata: { last_active: '2024-01-15T10:30:00Z' }
      },
      {
        name: 'Agent 2', 
        type: 'test',
        status: 'active',
        capabilities: [],
        metadata: { last_active: '2024-01-15T10:30:00Z' }
      },
      {
        name: 'Agent 3',
        type: 'test', 
        status: 'idle',
        capabilities: [],
        metadata: { last_active: '2024-01-15T10:30:00Z' }
      }
    ];

    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ agents: mockAgents }),
    } as Response);

    render(<AgentMonitor />);

    await waitFor(() => {
      // Should show 2 online (active) agents
      expect(screen.getByText('2')).toBeInTheDocument();
    });
  });

  it('should have refresh functionality that re-fetches data', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ agents: [] }),
    } as Response);

    render(<AgentMonitor />);

    // Wait for initial load
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    // Click refresh button
    const refreshButton = screen.getByText('Refresh Status');
    refreshButton.click();

    // Should make another API call
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });
  });
});
