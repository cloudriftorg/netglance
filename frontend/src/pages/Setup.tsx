import { FormEvent, useState } from 'react';
import { api } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';

export default function Setup({ onDone }: { onDone: () => void }) {
  const toast = useToast();
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (password.length < 8) {
      toast.error('Password too short (min 8 chars)');
      return;
    }
    if (password !== confirm) {
      toast.error('Passwords do not match');
      return;
    }
    setBusy(true);
    try {
      await api.setup({ username, password });
      toast.success('Welcome aboard');
      onDone();
    } catch (err) {
      toast.error(errMessage(err, 'Setup failed'));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-full items-center justify-center px-4 py-10">
      <form onSubmit={submit} className="w-full max-w-sm space-y-5 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <div>
          <h1 className="text-xl font-semibold">Welcome to Netglance</h1>
          <p className="mt-1 text-sm text-slate-500">Create the admin account to get started.</p>
        </div>
        <Field label="Username">
          <input value={username} onChange={(e) => setUsername(e.target.value)} required className="input" autoComplete="username" />
        </Field>
        <Field label="Password">
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} className="input" autoComplete="new-password" />
        </Field>
        <Field label="Confirm password">
          <input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required minLength={8} className="input" autoComplete="new-password" />
        </Field>
        <button type="submit" disabled={busy} className="btn-primary w-full">
          {busy ? 'Creating…' : 'Create admin'}
        </button>
        <p className="text-xs text-slate-500">After this you can configure networks, SMTP and gateway integration from Settings.</p>
      </form>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-1.5 text-sm">
      <span className="font-medium text-slate-700 dark:text-slate-300">{label}</span>
      {children}
    </label>
  );
}
