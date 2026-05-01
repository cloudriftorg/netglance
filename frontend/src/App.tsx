import { useEffect, useState } from 'react';
import { Navigate, Route, Routes, useNavigate } from 'react-router-dom';
import { api } from './lib/api';
import Setup from './pages/Setup';
import Login from './pages/Login';
import Hosts from './pages/Hosts';
import HostDetail from './pages/HostDetail';
import SettingsPage from './pages/Settings';
import Layout from './components/Layout';

type Auth = { state: 'loading' } | { state: 'guest'; setupComplete: boolean } | { state: 'authed'; username: string };

export default function App() {
  const [auth, setAuth] = useState<Auth>({ state: 'loading' });
  const navigate = useNavigate();

  async function refresh() {
    try {
      const me = await api.me();
      setAuth({ state: 'authed', username: me.username });
    } catch {
      const s = await api.setupStatus().catch(() => ({ setupComplete: false }));
      setAuth({ state: 'guest', setupComplete: s.setupComplete });
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  if (auth.state === 'loading') {
    return <div className="flex h-full items-center justify-center text-slate-500">Loading…</div>;
  }

  if (auth.state === 'guest') {
    return (
      <Routes>
        <Route
          path="/setup"
          element={auth.setupComplete ? <Navigate to="/login" replace /> : <Setup onDone={() => refresh()} />}
        />
        <Route
          path="/login"
          element={!auth.setupComplete ? <Navigate to="/setup" replace /> : <Login onDone={() => refresh()} />}
        />
        <Route path="*" element={<Navigate to={auth.setupComplete ? '/login' : '/setup'} replace />} />
      </Routes>
    );
  }

  return (
    <Layout
      username={auth.username}
      onLogout={async () => {
        await api.logout();
        navigate('/login');
        refresh();
      }}
    >
      <Routes>
        <Route path="/" element={<Hosts />} />
        <Route path="/h/:mac" element={<HostDetail />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  );
}
