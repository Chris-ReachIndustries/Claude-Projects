import { useState, useEffect } from 'react';
import { Routes, Route } from 'react-router-dom';
import { useAgents } from './hooks/useAgents';
import Sidebar from './components/Sidebar';
import Dashboard from './components/Dashboard';
import AgentDetail from './components/AgentDetail';
import Settings from './components/Settings';
import SetupWizard from './components/SetupWizard';
import Workflows from './components/Workflows';
import WorkflowDetail from './components/WorkflowDetail';
import WorkflowCreate from './components/WorkflowCreate';
import Projects from './components/Projects';
import ProjectDetail from './components/ProjectDetail';
import ErrorBoundary from './components/ErrorBoundary';
import { fetchSetupStatus, getStoredApiKey, setApiKey, fetchApiKey } from './api';

function App() {
  const { agents, loading, error, refetch, connectionState } = useAgents();
  const [setupComplete, setSetupComplete] = useState<boolean | null>(null);

  useEffect(() => {
    fetchSetupStatus()
      .then(async ({ complete }) => {
        setSetupComplete(complete);
        // Auto-fetch API key if not stored
        if (!getStoredApiKey()) {
          try {
            const key = await fetchApiKey();
            setApiKey(key);
            window.location.reload();
          } catch {
            // Will retry on next load
          }
        }
      })
      .catch(() => setSetupComplete(true));
  }, []);

  if (setupComplete === null) {
    return (
      <div className="min-h-screen bg-surface-ground flex items-center justify-center">
        <div className="flex items-center gap-3">
          <div className="w-2 h-2 bg-accent-400 rounded-full animate-pulse" />
          <span className="text-white/40 text-sm">Loading...</span>
        </div>
      </div>
    );
  }

  if (!setupComplete) {
    return <SetupWizard onComplete={() => setSetupComplete(true)} />;
  }

  const activeCount = agents.filter((a) =>
    ['active', 'working', 'idle', 'waiting-for-input'].includes(a.status)
  ).length;

  return (
    <div className="min-h-screen bg-surface-ground flex">
      <Sidebar
        agentCount={agents.length}
        activeCount={activeCount}
        connectionState={connectionState}
      />
      {/* Main content — offset by sidebar width */}
      <main className="flex-1 ml-52 min-h-screen">
        <Routes>
          <Route
            path="/"
            element={
              <Dashboard
                agents={agents}
                loading={loading}
                error={error}
                refetch={refetch}
              />
            }
          />
          <Route
            path="/agent/:id"
            element={
              <ErrorBoundary>
                <AgentDetail />
              </ErrorBoundary>
            }
          />
          <Route path="/settings" element={<Settings />} />
          <Route path="/projects" element={<Projects />} />
          <Route
            path="/projects/:id"
            element={
              <ErrorBoundary>
                <ProjectDetail />
              </ErrorBoundary>
            }
          />
          <Route path="/workflows" element={<Workflows />} />
          <Route path="/workflows/new" element={<WorkflowCreate />} />
          <Route path="/workflows/:id" element={<WorkflowDetail />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;
