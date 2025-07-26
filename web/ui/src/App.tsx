import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout/Layout';
import Dashboard from './components/Dashboard/Dashboard';
import GraphView from './components/GraphView/GraphView';
import ChatInterface from './components/ChatInterface/ChatInterface';
import AgentMonitor from './components/AgentMonitor/AgentMonitor';

function App() {
  return (
    <Router>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/graph" element={<GraphView />} />
          <Route path="/chat" element={<ChatInterface />} />
          <Route path="/agents" element={<AgentMonitor />} />
        </Routes>
      </Layout>
    </Router>
  );
}

export default App;
