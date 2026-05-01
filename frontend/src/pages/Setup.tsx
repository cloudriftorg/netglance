import { FormEvent, useEffect, useState } from 'react';
import { api, NetInterface } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';
import IfacePicker from '../components/IfacePicker';

export default function Setup({ onDone }: { onDone: () => void }) {
  const [step, setStep] = useState<'password' | 'interfaces'>('password');

  return step === 'password' ? (
    <PasswordStep onNext={() => setStep('interfaces')} />
  ) : (
    <InterfacesStep onDone={onDone} />
  );
}

function PasswordStep({ onNext }: { onNext: () => void }) {
  const toast = useToast();
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
      await api.setup({ password });
      onNext();
    } catch (err) {
      toast.error(errMessage(err, 'Setup failed'));
      setBusy(false);
    }
  }

  return (
    <Wizard step={1} title="Welcome to Netglance" subtitle="Set the admin password.">
      <form onSubmit={submit} className="space-y-5">
        <Field label="Password">
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={8}
            autoFocus
            className="input"
            autoComplete="new-password"
          />
        </Field>
        <Field label="Confirm password">
          <input
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            required
            minLength={8}
            className="input"
            autoComplete="new-password"
          />
        </Field>
        <button type="submit" disabled={busy} className="btn-primary w-full">
          {busy ? 'Saving…' : 'Continue'}
        </button>
      </form>
    </Wizard>
  );
}

function InterfacesStep({ onDone }: { onDone: () => void }) {
  const toast = useToast();
  const [ifaces, setIfaces] = useState<NetInterface[]>([]);
  const [scanIfaces, setScanIfaces] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.listInterfaces()
      .then((list) => {
        setIfaces(list);
        // Pre-select all detected interfaces — the most common case is "scan
        // everything". Empty selection is still valid and means no scan.
        setScanIfaces(list.map((i) => i.name));
      })
      .catch((e) => toast.error(errMessage(e, 'Could not load interfaces')))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      const current = await api.getSettings();
      await api.putSettings({ ...current, scanIfaces });
      toast.success('Welcome aboard');
      onDone();
    } catch (err) {
      toast.error(errMessage(err, 'Save failed'));
      setBusy(false);
    }
  }

  return (
    <Wizard
      step={2}
      title="Pick interfaces to scan"
      subtitle="Netglance will only ARP-scan the interfaces you select. You can change this later in Settings."
    >
      <form onSubmit={submit} className="space-y-5">
        {loading ? (
          <p className="text-sm text-slate-500">Loading interfaces…</p>
        ) : (
          <IfacePicker
            value={scanIfaces}
            ifaces={ifaces}
            onChange={setScanIfaces}
            emptyHelpText="Empty = no scan will run. Pick at least one interface to discover hosts."
          />
        )}
        <button type="submit" disabled={busy || loading} className="btn-primary w-full">
          {busy ? 'Saving…' : 'Finish'}
        </button>
      </form>
    </Wizard>
  );
}

function Wizard({
  step,
  title,
  subtitle,
  children,
}: {
  step: 1 | 2;
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-full items-center justify-center px-4 py-10">
      <div className="w-full max-w-md">
        <div className="space-y-5 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
          <div>
            <h1 className="text-xl font-semibold">{title}</h1>
            <p className="mt-1 text-sm text-slate-500">{subtitle}</p>
          </div>
          {children}
        </div>
        <div className="mt-4 flex items-center justify-center gap-2">
          <Pip active={step >= 1} done={step > 1} label="1" />
          <span className="h-px w-6 bg-slate-200 dark:bg-slate-700" />
          <Pip active={step >= 2} done={false} label="2" />
        </div>
      </div>
    </div>
  );
}

function Pip({ active, done, label }: { active: boolean; done: boolean; label: string }) {
  const cls = done
    ? 'bg-brand-500 text-white'
    : active
      ? 'border-2 border-brand-500 text-brand-600 dark:text-brand-300'
      : 'border border-slate-300 text-slate-400 dark:border-slate-700';
  return (
    <span className={`inline-flex h-7 w-7 items-center justify-center rounded-full text-xs font-semibold ${cls}`}>
      {done ? '✓' : label}
    </span>
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
