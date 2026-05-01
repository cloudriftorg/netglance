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
    request<{ username: string }>('/api/setup', { method: 'POST', body: JSON.stringify(body) }),
  login: (username: string, password: string) =>
    request<{ username: string }>('/api/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  logout: () => request<{ ok: boolean }>('/api/logout', { method: 'POST' }),
  me: () => request<{ username: string }>('/api/me'),

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

  runScan: () => request<{ status: string }>('/api/scan/run', { method: 'POST' }),
  scanStatus: () => request<{ running: boolean }>('/api/scan/status'),
  listScans: () => request<Scan[]>('/api/scans'),

  getSettings: () => request<Settings>('/api/settings'),
  putSettings: (s: Settings) => request<{ ok: boolean }>('/api/settings', { method: 'PUT', body: JSON.stringify(s) }),
  testSMTP: () => request<{ ok: boolean }>('/api/settings/test-smtp', { method: 'POST' }),
};

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
  id: number;
  startedAt: number;
  endedAt?: number;
  networkId?: string;
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
  offlineAfter: number;
  primaryIface: string;
  smtp?: SMTPConfig;
  notify: NotifyToggles;
}

export interface SetupBody {
  username: string;
  password: string;
  networks?: NetworkConfig[];
  smtp?: SMTPConfig;
  scanEverySeconds?: number;
}

export interface HostFilter {
  q?: string;
  vlan?: number | null;
  online?: boolean | null;
}
