/* ── Bootstrap ─────────────────────────────────────────────────────── */

if (!Auth.token) { window.location.href = 'index.html'; }

let currentRouterId = null;
let sseUnsubs = [];

/* ── User info ─────────────────────────────────────────────────────── */
const user = Auth.user;
if (user) {
  document.getElementById('user-name').textContent = user.username || '—';
  document.getElementById('user-role').textContent = user.role || '—';
  document.getElementById('user-avatar').textContent = (user.username || 'U')[0].toUpperCase();
}
document.getElementById('logout-btn').addEventListener('click', () => AuthAPI.logout());

/* ── Routing (tab nav) ─────────────────────────────────────────────── */
const TITLES = {
  'dashboard':      'Dashboard',
  'hotspot-users':  'Hotspot › Users',
  'hotspot-active': 'Hotspot › Active',
  'ppp-secrets':    'PPP › Secrets',
  'ppp-active':     'PPP › Active Sessions',
};

document.querySelectorAll('.nav-item[data-view]').forEach(el => {
  el.addEventListener('click', e => {
    e.preventDefault();
    navigate(el.dataset.view);
  });
});

function navigate(view) {
  document.querySelectorAll('.nav-item').forEach(n => n.classList.toggle('active', n.dataset.view === view));
  document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
  document.getElementById(`view-${view}`)?.classList.add('active');
  document.getElementById('page-title').textContent = TITLES[view] || view;
  onViewEnter(view);
}

function onViewEnter(view) {
  if (!currentRouterId) return;
  if (view === 'dashboard')      loadDashboard();
  if (view === 'hotspot-users')  loadHotspotUsers();
  if (view === 'hotspot-active') loadHotspotActive();
  if (view === 'ppp-secrets')    loadPPPSecrets();
  if (view === 'ppp-active')     loadPPPActive();
}

/* ── Router select ─────────────────────────────────────────────────── */
const routerSelect = document.getElementById('router-select');

(async () => {
  try {
    const routers = await Routers.list();
    if (!routers?.length) return;
    routers.forEach(r => {
      const opt = document.createElement('option');
      opt.value = r.id;
      opt.textContent = `${r.session_name} (${r.ip})`;
      routerSelect.appendChild(opt);
    });
    // Auto-select first
    routerSelect.value = routers[0].id;
    selectRouter(routers[0].id, `${routers[0].session_name} (${routers[0].ip})`);
  } catch (err) { console.error('Failed to load routers', err); }
})();

routerSelect.addEventListener('change', () => {
  const opt = routerSelect.selectedOptions[0];
  selectRouter(routerSelect.value, opt.textContent);
});

function selectRouter(id, label) {
  currentRouterId = id || null;
  document.getElementById('router-badge').textContent = label || 'No router selected';

  // Tear down existing SSE subs
  sseUnsubs.forEach(fn => fn());
  sseUnsubs = [];

  if (!id) return;

  // Start live log stream
  startLogStream(id);
  // Start SSE resource monitor
  startResourceSSE(id);

  // Re-load current view
  const activeNav = document.querySelector('.nav-item.active');
  if (activeNav) onViewEnter(activeNav.dataset.view);
}

/* ── Dashboard ─────────────────────────────────────────────────────── */
async function loadDashboard() {
  try {
    const [res, identity] = await Promise.all([
      System.resource(currentRouterId).catch(() => null),
      System.identity(currentRouterId).catch(() => null),
    ]);

    if (res) updateResourceUI(res);
    if (identity) {
      // identity is map[string]string with key "name"
      document.getElementById('si-identity').textContent = identity.name || identity.Name || '—';
    }

    // Counts via list length (small routers, acceptable)
    const [hsUsers, hsActive, pppSecrets, pppActive] = await Promise.all([
      Hotspot.users(currentRouterId).catch(() => null),
      Hotspot.active(currentRouterId).catch(() => null),
      PPP.secrets(currentRouterId).catch(() => null),
      PPP.active(currentRouterId).catch(() => null),
    ]);

    setText('stat-hs-users',   Array.isArray(hsUsers)    ? hsUsers.length    : '—');
    setText('stat-hs-active',  Array.isArray(hsActive)   ? hsActive.length   : '—');
    setText('stat-ppp-secrets',Array.isArray(pppSecrets) ? pppSecrets.length : '—');
    setText('stat-ppp-active', Array.isArray(pppActive)  ? pppActive.length  : '—');
  } catch (err) { console.error('Dashboard error', err); }
}

function updateResourceUI(res) {
  // REST returns map[string]string — all values are strings, keys are snake_case
  const cpu      = Number(res.cpu_load || 0);
  const freeMem  = Number(res.free_memory || 0);
  const totalMem = Number(res.total_memory || 0);
  const memPct   = totalMem ? Math.round((1 - freeMem / totalMem) * 100) : 0;

  setText('stat-cpu', `${cpu}%`);
  setText('stat-mem', `${memPct}%`);
  document.getElementById('bar-cpu').style.width = `${Math.min(cpu, 100)}%`;
  document.getElementById('bar-mem').style.width = `${Math.min(memPct, 100)}%`;

  if (res.uptime)            setText('si-uptime',  res.uptime);
  if (res.version)           setText('si-version', res.version);
  if (res.board_name)        setText('si-board',   res.board_name);
  if (res.architecture_name) setText('si-arch',    res.architecture_name);
}

function startResourceSSE(routerId) {
  // system_resource is now a stream command (=interval=1s) — fields are included.
  const unsub = SSE.subscribe(routerId, 'system/resource', (data) => {
    updateSSEStatus(true);
    const fields = data.fields;
    if (fields && Object.keys(fields).length > 0) {
      updateResourceUI(fields);
    }
  });
  sseUnsubs.push(unsub);
}

/* ── Log stream ────────────────────────────────────────────────────── */
function startLogStream(routerId) {
  const container = document.getElementById('log-stream');
  container.innerHTML = '';
  const MAX_ENTRIES = 200;

  const unsub = SSE.subscribe(routerId, 'logs/all', (data) => {
    updateSSEStatus(true);
    const entry = document.createElement('div');
    entry.className = 'log-entry';

    // data may be a string line or a JSON object
    if (typeof data === 'string') {
      entry.textContent = data;
    } else {
      const time  = data.time  || data.timestamp || '';
      const topic = data.topics || data.topic || '';
      const msg   = data.message || data.msg || JSON.stringify(data);
      entry.innerHTML = `<span class="log-time">${time.slice(11,19)}</span><span class="log-topic">[${topic}]</span>${escapeHtml(msg)}`;
    }

    container.appendChild(entry);
    // Trim old entries
    while (container.children.length > MAX_ENTRIES) container.removeChild(container.firstChild);
    container.scrollTop = container.scrollHeight;
  });
  sseUnsubs.push(unsub);
}

/* ── Hotspot Users ─────────────────────────────────────────────────── */
let _hsUsers = [];

async function loadHotspotUsers() {
  const tbody = document.getElementById('hs-user-tbody');
  tbody.innerHTML = '<tr><td colspan="8" style="color:var(--text-muted);text-align:center;padding:20px">Loading…</td></tr>';
  try {
    _hsUsers = await Hotspot.users(currentRouterId) || [];
    renderHotspotUsers(_hsUsers);

    // Start live SSE for users
    clearSSEByKey(`${currentRouterId}:hotspot/users`);
    const unsub = SSE.subscribe(currentRouterId, 'hotspot/users', (data) => {
      updateSSEStatus(true);
      if (!document.getElementById('view-hotspot-users').classList.contains('active')) return;
      // Merge updated record
      const fields = data.fields || data;
      const name   = data.tags?.name || fields.name;
      if (!name) return;
      const idx = _hsUsers.findIndex(u => u.name === name);
      if (idx >= 0) { _hsUsers[idx] = { ..._hsUsers[idx], ...fields }; }
      else { _hsUsers.push(fields); }
      renderHotspotUsers(_hsUsers);
    });
    sseUnsubs.push(unsub);
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="8" style="color:var(--red);padding:14px">${err.message}</td></tr>`;
  }
}

function renderHotspotUsers(users) {
  const q = document.getElementById('hs-user-search').value.toLowerCase();
  const filtered = q ? users.filter(u => (u.name || '').toLowerCase().includes(q)) : users;
  const tbody = document.getElementById('hs-user-tbody');
  const empty = document.getElementById('hs-user-empty');

  if (!filtered.length) {
    tbody.innerHTML = '';
    empty.classList.remove('hidden');
    return;
  }
  empty.classList.add('hidden');

  tbody.innerHTML = filtered.map(u => {
    const disabled = u.disabled === 'true' || u.disabled === true;
    return `<tr>
      <td>${escapeHtml(u.name || '—')}</td>
      <td>${escapeHtml(u.profile || '—')}</td>
      <td>${escapeHtml(u.server || '—')}</td>
      <td>${escapeHtml(u.uptime || '0s')}</td>
      <td>${formatBytes(u['bytes-in'] ?? u.bytes_in ?? 0)}</td>
      <td>${formatBytes(u['bytes-out'] ?? u.bytes_out ?? 0)}</td>
      <td><span class="badge ${disabled ? 'badge-red' : 'badge-green'}">${disabled ? 'Disabled' : 'Active'}</span></td>
      <td><div class="actions">
        <button class="btn-table" onclick="editHotspotUser('${escapeHtml(u['.id'] || u.id || '')}')">Edit</button>
        <button class="btn-table danger" onclick="deleteHotspotUser('${escapeHtml(u['.id'] || u.id || '')}', '${escapeHtml(u.name || '')}')">Delete</button>
      </div></td>
    </tr>`;
  }).join('');
}

document.getElementById('hs-user-search').addEventListener('input', () => renderHotspotUsers(_hsUsers));

document.getElementById('hs-user-add-btn').addEventListener('click', () => {
  openModal('Add Hotspot User', hotspotUserForm({}), async () => {
    const data = collectForm('hs-user-form');
    await Hotspot.createUser(currentRouterId, data);
    await loadHotspotUsers();
  });
});

function editHotspotUser(id) {
  const u = _hsUsers.find(x => (x['.id'] || x.id) === id);
  if (!u) return;
  openModal('Edit Hotspot User', hotspotUserForm(u), async () => {
    const data = collectForm('hs-user-form');
    await Hotspot.updateUser(currentRouterId, id, data);
    await loadHotspotUsers();
  });
}

function deleteHotspotUser(id, name) {
  showConfirm(`Delete hotspot user "${name}"?`, async () => {
    await Hotspot.deleteUser(currentRouterId, id);
    await loadHotspotUsers();
  });
}

function hotspotUserForm(u) {
  return `<form id="hs-user-form">
    <div class="field-row">
      <div class="field"><label>Name</label><input name="name" value="${escapeHtml(u.name||'')}" required /></div>
      <div class="field"><label>Password</label><input name="password" type="text" value="${escapeHtml(u.password||'')}" /></div>
    </div>
    <div class="field-row">
      <div class="field"><label>Profile</label><input name="profile" value="${escapeHtml(u.profile||'')}" /></div>
      <div class="field"><label>Server</label><input name="server" value="${escapeHtml(u.server||'')}" /></div>
    </div>
    <div class="field-row">
      <div class="field"><label>Limit Uptime</label><input name="limit-uptime" value="${escapeHtml(u['limit-uptime']||'')}" placeholder="24h" /></div>
      <div class="field"><label>Limit Bytes Total</label><input name="limit-bytes-total" value="${escapeHtml(u['limit-bytes-total']||'')}" placeholder="1G" /></div>
    </div>
    <div class="field"><label>Comment</label><input name="comment" value="${escapeHtml(u.comment||'')}" /></div>
  </form>`;
}

/* ── Hotspot Active ────────────────────────────────────────────────── */
let _hsActive = [];

async function loadHotspotActive() {
  const tbody = document.getElementById('hs-active-tbody');
  tbody.innerHTML = '<tr><td colspan="7" style="color:var(--text-muted);text-align:center;padding:20px">Loading…</td></tr>';
  try {
    _hsActive = await Hotspot.active(currentRouterId) || [];
    renderHotspotActive(_hsActive);

    const unsub = SSE.subscribe(currentRouterId, 'hotspot/active', (data) => {
      updateSSEStatus(true);
      if (!document.getElementById('view-hotspot-active').classList.contains('active')) return;
      const fields = data.fields || data;
      const id = data.tags?.id || fields['.id'] || fields.id;
      if (data.type === 'delete') {
        _hsActive = _hsActive.filter(s => (s['.id'] || s.id) !== id);
      } else {
        const idx = _hsActive.findIndex(s => (s['.id'] || s.id) === id);
        if (idx >= 0) _hsActive[idx] = { ..._hsActive[idx], ...fields };
        else _hsActive.push(fields);
      }
      renderHotspotActive(_hsActive);
    });
    sseUnsubs.push(unsub);
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="7" style="color:var(--red);padding:14px">${err.message}</td></tr>`;
  }
}

function renderHotspotActive(sessions) {
  const tbody = document.getElementById('hs-active-tbody');
  const empty = document.getElementById('hs-active-empty');

  if (!sessions.length) {
    tbody.innerHTML = '';
    empty.classList.remove('hidden');
    return;
  }
  empty.classList.add('hidden');

  tbody.innerHTML = sessions.map(s => `<tr>
    <td>${escapeHtml(s.user || s.User || '—')}</td>
    <td>${escapeHtml(s.address || s.Address || '—')}</td>
    <td>${escapeHtml(s['mac-address'] || s.MACAddress || '—')}</td>
    <td>${escapeHtml(s.server || s.Server || '—')}</td>
    <td>${escapeHtml(s.uptime || s.Uptime || '—')}</td>
    <td>${escapeHtml(s['login-by'] || s.LoginBy || '—')}</td>
    <td><div class="actions">
      <button class="btn-table danger" onclick="disconnectHotspot('${escapeHtml(s['.id']||s.id||'')}', '${escapeHtml(s.user||s.User||'')}')">Disconnect</button>
    </div></td>
  </tr>`).join('');
}

function disconnectHotspot(id, user) {
  showConfirm(`Disconnect hotspot user "${user}"?`, async () => {
    await Hotspot.disconnect(currentRouterId, id);
    await loadHotspotActive();
  });
}

/* ── PPP Secrets ───────────────────────────────────────────────────── */
let _pppSecrets = [];
let _pppProfiles = [];

async function loadPPPSecrets() {
  const tbody = document.getElementById('ppp-secret-tbody');
  tbody.innerHTML = '<tr><td colspan="6" style="color:var(--text-muted);text-align:center;padding:20px">Loading…</td></tr>';
  try {
    [_pppSecrets, _pppProfiles] = await Promise.all([
      PPP.secrets(currentRouterId).catch(() => []),
      PPP.profiles(currentRouterId).catch(() => []),
    ]);
    renderPPPSecrets(_pppSecrets);

    const unsub = SSE.subscribe(currentRouterId, 'ppp/secrets', (data) => {
      updateSSEStatus(true);
      if (!document.getElementById('view-ppp-secrets').classList.contains('active')) return;
      const fields = data.fields || data;
      const name = data.tags?.name || fields.name;
      if (!name) return;
      const idx = _pppSecrets.findIndex(s => s.name === name);
      if (idx >= 0) _pppSecrets[idx] = { ..._pppSecrets[idx], ...fields };
      else _pppSecrets.push(fields);
      renderPPPSecrets(_pppSecrets);
    });
    sseUnsubs.push(unsub);
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="6" style="color:var(--red);padding:14px">${err.message}</td></tr>`;
  }
}

function renderPPPSecrets(secrets) {
  const q = document.getElementById('ppp-secret-search').value.toLowerCase();
  const filtered = q ? secrets.filter(s => (s.name || '').toLowerCase().includes(q)) : secrets;
  const tbody = document.getElementById('ppp-secret-tbody');
  const empty = document.getElementById('ppp-secret-empty');

  if (!filtered.length) {
    tbody.innerHTML = '';
    empty.classList.remove('hidden');
    return;
  }
  empty.classList.add('hidden');

  tbody.innerHTML = filtered.map(s => {
    const disabled = s.disabled === 'true' || s.disabled === true;
    return `<tr>
      <td>${escapeHtml(s.name || '—')}</td>
      <td>${escapeHtml(s.service || '—')}</td>
      <td>${escapeHtml(s.profile || '—')}</td>
      <td>${escapeHtml(s['client-address'] || '—')}</td>
      <td><span class="badge ${disabled ? 'badge-red' : 'badge-green'}">${disabled ? 'Disabled' : 'Active'}</span></td>
      <td><div class="actions">
        <button class="btn-table" onclick="editPPPSecret('${escapeHtml(s['.id']||s.id||'')}')">Edit</button>
        <button class="btn-table danger" onclick="deletePPPSecret('${escapeHtml(s['.id']||s.id||'')}', '${escapeHtml(s.name||'')}')">Delete</button>
      </div></td>
    </tr>`;
  }).join('');
}

document.getElementById('ppp-secret-search').addEventListener('input', () => renderPPPSecrets(_pppSecrets));

document.getElementById('ppp-secret-add-btn').addEventListener('click', () => {
  openModal('Add PPP Secret', pppSecretForm({}), async () => {
    const data = collectForm('ppp-secret-form');
    await PPP.createSecret(currentRouterId, data);
    await loadPPPSecrets();
  });
});

function editPPPSecret(id) {
  const s = _pppSecrets.find(x => (x['.id'] || x.id) === id);
  if (!s) return;
  openModal('Edit PPP Secret', pppSecretForm(s), async () => {
    const data = collectForm('ppp-secret-form');
    await PPP.updateSecret(currentRouterId, id, data);
    await loadPPPSecrets();
  });
}

function deletePPPSecret(id, name) {
  showConfirm(`Delete PPP secret "${name}"?`, async () => {
    await PPP.deleteSecret(currentRouterId, id);
    await loadPPPSecrets();
  });
}

function pppSecretForm(s) {
  const profileOpts = _pppProfiles.map(p => `<option value="${escapeHtml(p.name||p)}" ${s.profile===p.name?'selected':''}>${escapeHtml(p.name||p)}</option>`).join('');
  const services = ['any','pptp','l2tp','sstp','pppoe','ovpn'].map(v =>
    `<option value="${v}" ${s.service===v?'selected':''}>${v}</option>`).join('');
  return `<form id="ppp-secret-form">
    <div class="field-row">
      <div class="field"><label>Name</label><input name="name" value="${escapeHtml(s.name||'')}" required /></div>
      <div class="field"><label>Password</label><input name="password" type="text" value="${escapeHtml(s.password||'')}" /></div>
    </div>
    <div class="field-row">
      <div class="field"><label>Service</label><select name="service">${services}</select></div>
      <div class="field"><label>Profile</label><select name="profile"><option value="">—</option>${profileOpts}</select></div>
    </div>
    <div class="field-row">
      <div class="field"><label>Client IP</label><input name="client-address" value="${escapeHtml(s['client-address']||'')}" placeholder="0.0.0.0" /></div>
      <div class="field"><label>Remote IP</label><input name="server-address" value="${escapeHtml(s['server-address']||'')}" placeholder="0.0.0.0" /></div>
    </div>
    <div class="field"><label>Comment</label><input name="comment" value="${escapeHtml(s.comment||'')}" /></div>
  </form>`;
}

/* ── PPP Active ────────────────────────────────────────────────────── */
let _pppActive = [];

async function loadPPPActive() {
  const tbody = document.getElementById('ppp-active-tbody');
  tbody.innerHTML = '<tr><td colspan="6" style="color:var(--text-muted);text-align:center;padding:20px">Loading…</td></tr>';
  try {
    _pppActive = await PPP.active(currentRouterId) || [];
    renderPPPActive(_pppActive);

    const unsub = SSE.subscribe(currentRouterId, 'ppp/active', (data) => {
      updateSSEStatus(true);
      if (!document.getElementById('view-ppp-active').classList.contains('active')) return;
      const fields = data.fields || data;
      const id = data.tags?.id || fields['.id'] || fields.id;
      if (data.type === 'delete') {
        _pppActive = _pppActive.filter(s => (s['.id'] || s.id) !== id);
      } else {
        const idx = _pppActive.findIndex(s => (s['.id'] || s.id) === id);
        if (idx >= 0) _pppActive[idx] = { ..._pppActive[idx], ...fields };
        else _pppActive.push(fields);
      }
      renderPPPActive(_pppActive);
    });
    sseUnsubs.push(unsub);
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="6" style="color:var(--red);padding:14px">${err.message}</td></tr>`;
  }
}

function renderPPPActive(sessions) {
  const tbody = document.getElementById('ppp-active-tbody');
  const empty = document.getElementById('ppp-active-empty');

  if (!sessions.length) {
    tbody.innerHTML = '';
    empty.classList.remove('hidden');
    return;
  }
  empty.classList.add('hidden');

  tbody.innerHTML = sessions.map(s => `<tr>
    <td>${escapeHtml(s.name || s.Name || '—')}</td>
    <td>${escapeHtml(s.service || s.Service || '—')}</td>
    <td>${escapeHtml(s.address || s.Address || '—')}</td>
    <td>${escapeHtml(s['remote-address'] || s.RemoteAddress || '—')}</td>
    <td>${escapeHtml(s.uptime || s.Uptime || '—')}</td>
    <td><div class="actions">
      <button class="btn-table danger" onclick="disconnectPPP('${escapeHtml(s['.id']||s.id||'')}', '${escapeHtml(s.name||s.Name||'')}')">Disconnect</button>
    </div></td>
  </tr>`).join('');
}

function disconnectPPP(id, name) {
  showConfirm(`Disconnect PPP session "${name}"?`, async () => {
    await PPP.disconnect(currentRouterId, id);
    await loadPPPActive();
  });
}

/* ── Modal ─────────────────────────────────────────────────────────── */
let _modalSave = null;

function openModal(title, bodyHTML, onSave) {
  document.getElementById('modal-title').textContent = title;
  document.getElementById('modal-body').innerHTML = bodyHTML;
  document.getElementById('modal-overlay').classList.remove('hidden');
  _modalSave = onSave;
}

document.getElementById('modal-close').addEventListener('click',  closeModal);
document.getElementById('modal-cancel').addEventListener('click', closeModal);

document.getElementById('modal-save').addEventListener('click', async () => {
  if (!_modalSave) return;
  const btn = document.getElementById('modal-save');
  btn.disabled = true;
  btn.textContent = 'Saving…';
  try {
    await _modalSave();
    closeModal();
  } catch (err) {
    alert(`Error: ${err.message}`);
  } finally {
    btn.disabled = false;
    btn.textContent = 'Save';
  }
});

function closeModal() {
  document.getElementById('modal-overlay').classList.add('hidden');
  _modalSave = null;
}

document.getElementById('modal-overlay').addEventListener('click', e => {
  if (e.target === e.currentTarget) closeModal();
});

/* ── Confirm dialog ────────────────────────────────────────────────── */
let _confirmCb = null;

function showConfirm(text, cb) {
  document.getElementById('confirm-text').textContent = text;
  document.getElementById('confirm-overlay').classList.remove('hidden');
  _confirmCb = cb;
}

document.getElementById('confirm-cancel').addEventListener('click', () => {
  document.getElementById('confirm-overlay').classList.add('hidden');
  _confirmCb = null;
});

document.getElementById('confirm-ok').addEventListener('click', async () => {
  document.getElementById('confirm-overlay').classList.add('hidden');
  if (_confirmCb) { try { await _confirmCb(); } catch(e) { alert(`Error: ${e.message}`); } }
  _confirmCb = null;
});

/* ── Utilities ─────────────────────────────────────────────────────── */
function setText(id, val) {
  const el = document.getElementById(id);
  if (el) el.textContent = val;
}

function escapeHtml(str) {
  return String(str).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function collectForm(formId) {
  const form = document.getElementById(formId);
  const data = {};
  new FormData(form).forEach((v, k) => { if (v !== '') data[k] = v; });
  return data;
}

function updateSSEStatus(ok) {
  const el = document.getElementById('sse-status');
  el.className = 'sse-status ' + (ok ? 'connected' : 'error');
}

function clearSSEByKey(key) {
  // No direct access needed; SSE manager handles dedup
}

/* ── Init ──────────────────────────────────────────────────────────── */
navigate('dashboard');
