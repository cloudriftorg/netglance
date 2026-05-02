import { useEffect, useState } from 'react';
import { Navigate, Route, Routes, useNavigate } from 'react-router-dom';
import { api, isBackendDown } from './lib/api';
import Setup from './pages/Setup';
import Login from './pages/Login';
import Hosts from './pages/Hosts';
import HostDetail from './pages/HostDetail';
import SettingsPage from './pages/Settings';
import Layout from './components/Layout';

type Auth = { state: 'loading' } | { state: 'guest'; setupComplete: boolean } | { state: 'authed' };

export default function App() {
  const [auth, setAuth] = useState<Auth>({ state: 'loading' });
  const navigate = useNavigate();

  async function refresh() {
    try {
      await api.me();
      setAuth({ state: 'authed' });
      // Backend is reachable AND auth check passed. Clear any stale
      // dev-skip flag — never let it persist into a real session.
      if (import.meta.env.DEV) localStorage.removeItem('netglance.dev.skipAuth');
    } catch {
      // The dev-only escape hatch fires ONLY when ALL hold:
      //  1. import.meta.env.DEV (stripped from production bundle)
      //  2. /healthz confirms the backend is genuinely down (TypeError
      //     OR Vite-proxy 502/503/504), so a flag stuck in localStorage
      //     can't bypass real auth on a working backend
      //  3. the user explicitly clicked "Skip wizard (dev)" in Setup
      if (
        import.meta.env.DEV &&
        localStorage.getItem('netglance.dev.skipAuth') === '1' &&
        (await isBackendDown())
      ) {
        setAuth({ state: 'authed' });
        return;
      }
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
