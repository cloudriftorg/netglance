import { FormEvent, useState } from 'react';
import { api } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';

export default function Login({ onDone }: { onDone: () => void }) {
  const toast = useToast();
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.login(password);
      onDone();
    } catch (err) {
      toast.error(errMessage(err, 'Login failed'));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-full items-center justify-center px-4 py-10">
      <form onSubmit={submit} className="w-full max-w-sm space-y-5 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <h1 className="text-xl font-semibold">Sign in</h1>
        <label className="block space-y-1.5 text-sm">
          <span className="font-medium text-slate-700 dark:text-slate-300">Password</span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoFocus
            className="input"
            autoComplete="current-password"
          />
        </label>
        <button type="submit" disabled={busy} className="btn-primary w-full">
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  );
}
