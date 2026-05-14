/* API client with automatic JWT refresh */

const API_BASE = 'http://localhost:8080/api/v1';

const Auth = {
  get token() { return localStorage.getItem('access_token'); },
  get refresh() { return localStorage.getItem('refresh_token'); },
  get user() {
    try { return JSON.parse(localStorage.getItem('user') || 'null'); }
    catch { return null; }
  },
  set(data) {
    localStorage.setItem('access_token', data.access_token);
    localStorage.setItem('refresh_token', data.refresh_token);
    if (data.user) localStorage.setItem('user', JSON.stringify(data.user));
  },
  clear() {
    ['access_token', 'refresh_token', 'user'].forEach(k => localStorage.removeItem(k));
  }
};

let _refreshing = null;

async function _doRefresh() {
  const res = await fetch(`${API_BASE}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: Auth.refresh })
  });
  if (!res.ok) throw new Error('refresh_failed');
  const body = await res.json();
  Auth.set(body.data);
}

async function request(path, opts = {}) {
  if (!Auth.token) { window.location.href = 'index.html'; return; }

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${Auth.token}`,
    ...(opts.headers || {})
  };

  let res = await fetch(`${API_BASE}${path}`, { ...opts, headers });

  if (res.status === 401) {
    // Try refresh once
    if (!_refreshing) _refreshing = _doRefresh().catch(() => { Auth.clear(); window.location.href = 'index.html'; });
    await _refreshing;
    _refreshing = null;
    headers['Authorization'] = `Bearer ${Auth.token}`;
    res = await fetch(`${API_BASE}${path}`, { ...opts, headers });
  }

  if (res.status === 204) return null;

  const body = await res.json();
  if (!res.ok) throw new Error(body.error || `HTTP ${res.status}`);
  return body.data;
}

const get  = (path)         => request(path, { method: 'GET' });
const del  = (path)         => request(path, { method: 'DELETE' });
const post = (path, data)   => request(path, { method: 'POST',   body: JSON.stringify(data) });
const put  = (path, data)   => request(path, { method: 'PUT',    body: JSON.stringify(data) });

/* Routers */
const Routers = {
  list: ()   => get('/routers'),
  get:  (id) => get(`/routers/${id}`),
};

/* Hotspot */
const Hotspot = {
  users:        (rid)         => get(`/routers/${rid}/hotspot/users`),
  userCount:    (rid)         => get(`/routers/${rid}/hotspot/users/count`),
  createUser:   (rid, data)   => post(`/routers/${rid}/hotspot/users`, data),
  updateUser:   (rid, id, d)  => put(`/routers/${rid}/hotspot/users/${id}`, d),
  deleteUser:   (rid, id)     => del(`/routers/${rid}/hotspot/users/${id}`),
  active:       (rid)         => get(`/routers/${rid}/hotspot/active`),
  disconnect:   (rid, id)     => post(`/routers/${rid}/hotspot/active/${id}/disconnect`, {}),
  inactive:     (rid)         => get(`/routers/${rid}/hotspot/inactive`),
  inactiveCount:(rid)         => get(`/routers/${rid}/hotspot/inactive/count`),
};

/* PPP */
const PPP = {
  secrets:      (rid)         => get(`/routers/${rid}/ppp/secrets`),
  createSecret: (rid, data)   => post(`/routers/${rid}/ppp/secrets`, data),
  updateSecret: (rid, id, d)  => put(`/routers/${rid}/ppp/secrets/${id}`, d),
  deleteSecret: (rid, id)     => del(`/routers/${rid}/ppp/secrets/${id}`),
  active:       (rid)         => get(`/routers/${rid}/ppp/active`),
  disconnect:   (rid, id)     => del(`/routers/${rid}/ppp/active/${id}`),
  profiles:     (rid)         => get(`/routers/${rid}/ppp/profiles`),
  inactive:     (rid)         => get(`/routers/${rid}/ppp/inactive`),
  inactiveCount:(rid)         => get(`/routers/${rid}/ppp/inactive/count`),
};

/* System */
const System = {
  resource: (rid) => get(`/routers/${rid}/system/resource`),
  identity: (rid) => get(`/routers/${rid}/system/identity`),
};

/* Auth */
const AuthAPI = {
  logout: () => post('/auth/logout', { refresh_token: Auth.refresh }).finally(() => { Auth.clear(); window.location.href = 'index.html'; }),
};

/* Helpers */
function formatBytes(n) {
  n = Number(n);
  if (!n || isNaN(n)) return '0 B';
  const u = ['B','KB','MB','GB','TB'];
  const i = Math.floor(Math.log(n) / Math.log(1024));
  return (n / Math.pow(1024, i)).toFixed(i ? 1 : 0) + ' ' + u[i];
}
