async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(text || `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json')) return res.json();
  return (await res.text()) as unknown as T;
}

export const api = {
  setupStatus: () => request<{ setupComplete: boolean }>('/api/setup/status'),
  setup: (body: SetupBody) =>
    request<{ ok: boolean }>('/api/setup', { method: 'POST', body: JSON.stringify(body) }),
  login: (password: string) =>
    request<{ ok: boolean }>('/api/login', { method: 'POST', body: JSON.stringify({ password }) }),
  logout: () => request<{ ok: boolean }>('/api/logout', { method: 'POST' }),
  me: () => request<{ ok: boolean }>('/api/me'),

  listHosts: (q?: HostFilter) => {
    const params = new URLSearchParams();
    if (q?.q) params.set('q', q.q);
    if (q?.vlan != null) params.set('vlan', String(q.vlan));
    if (q?.online != null) params.set('online', q.online ? '1' : '0');
    const qs = params.toString();
    return request<Host[]>(`/api/hosts${qs ? `?${qs}` : ''}`);
  },
  getHost: (mac: string) => request<{ host: Host; events: HostEvent[] }>(`/api/hosts/${mac}`),
  updateHost: (
    mac: string,
    body: { customName: string; customVendor: string; notifyOffline: boolean; isNew: boolean }
  ) => request<Host>(`/api/hosts/${mac}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteHost: (mac: string) => request<{ ok: boolean }>(`/api/hosts/${mac}`, { method: 'DELETE' }),

  runScan: () => request<{ status: string }>('/api/scan/run', { method: 'POST' }),
  scanStatus: () => request<{ running: boolean; lastScan?: Scan }>('/api/scan/status'),

  getSettings: () => request<Settings>('/api/settings'),
  putSettings: (s: Settings) => request<{ ok: boolean }>('/api/settings', { method: 'PUT', body: JSON.stringify(s) }),
  testSMTP: () => request<{ ok: boolean }>('/api/settings/test-smtp', { method: 'POST' }),

  listInterfaces: () => request<NetInterface[]>('/api/system/interfaces'),

  resetApp: (password: string) =>
    request<{ ok: boolean }>('/api/admin/reset', { method: 'POST', body: JSON.stringify({ password }) }),
};

export interface NetInterface {
  name: string;
  addresses: string[];
}

export interface Host {
  id: number;
  mac: string;
  ip: string;
  vlanId?: number;
  networkName?: string;
  hostname?: string;
  vendor?: string;
  customVendor?: string;
  customName?: string;
  firstSeen: number;
  lastSeen: number;
  online: boolean;
  isNew: boolean;
  notifyOffline: boolean;
}

export interface HostEvent {
  id: number;
  hostId: number;
  ts: number;
  kind: 'new' | 'online' | 'offline' | 'ip_change';
  ip?: string;
}

export interface Scan {
  startedAt: number;
  endedAt: number;
  hostsFound: number;
  error?: string;
}

export interface NetworkConfig {
  name: string;
  cidr: string;
  vlanId?: number;
}

export interface SMTPConfig {
  host: string;
  port: number;
  useTLS: boolean;
  useAuth: boolean;
  username?: string;
  password?: string;
  from: string;
  recipients: string[];
}

export interface NotifyToggles {
  newHost: boolean;
  offline: boolean;
  backOnline: boolean;
}

export interface Settings {
  networks: NetworkConfig[];
  scanEnabled: boolean;
  scanEverySeconds: number;
  scanIfaces: string[];
  offlineAfter: number;
  smtp?: SMTPConfig;
  notify: NotifyToggles;
}

export interface SetupBody {
  password: string;
}

export interface HostFilter {
  q?: string;
  vlan?: number | null;
  online?: boolean | null;
}
