import { render, screen, waitFor } from '@testing-library/react';
import Dashboard from './Dashboard';

// Mock fetch globally
const mockFetch = jest.fn();
global.fetch = mockFetch as any;

describe('Dashboard', () => {
  beforeEach(() => {
    mockFetch.mockClear();
  });

  it('should render dashboard with static content', () => {
    render(<Dashboard />);
    
    expect(screen.getByTestId('dashboard-view')).toBeInTheDocument();
    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(screen.getByText('NeuroMesh AI Orchestration Platform')).toBeInTheDocument();
  });

  it('should fetch and display agent status from API', async () => {
    // Mock the API response
    const mockAgentResponse = {
      agents: [
        {
          name: 'text-processor',
          type: 'processing',
          status: 'active',
          capabilities: ['text_analysis', 'nlp_processing'],
          metadata: { last_active: '2025-07-26T10:00:00Z' }
        },
        {
          name: 'data-analyzer',
          type: 'analysis',
          status: 'active',
          capabilities: ['data_processing', 'statistical_analysis'],
          metadata: { last_active: '2025-07-26T09:58:00Z' }
        }
      ]
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockAgentResponse,
    });

    render(<Dashboard />);

    // Wait for API call and UI update
    await waitFor(() => {
      expect(screen.getByText('2')).toBeInTheDocument(); // Active agents count
    });

    expect(mockFetch).toHaveBeenCalledWith('http://localhost:8081/api/agents/status');
  });

  it('should handle API errors gracefully', async () => {
    // Mock API error
    mockFetch.mockRejectedValueOnce(new Error('API Error'));

    render(<Dashboard />);

    // Should still render the dashboard with default values
    await waitFor(() => {
      expect(screen.getByTestId('dashboard-view')).toBeInTheDocument();
    });
  });

  it('should fetch system health status', async () => {
    // Mock health check response
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ status: 'ok', service: 'conversation-aware-web-bff' }),
    });

    render(<Dashboard />);

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('http://localhost:8081/health');
    });
  });
});
