/* SSE manager — reconnects automatically with exponential backoff */

const SSE_BASE = 'http://localhost:8080/api/v1';

class SSEManager {
  constructor() {
    this._sources  = {};  // key -> EventSource
    this._handlers = {};  // key -> Set<fn>
    this._retries  = {};  // key -> retry count
    this._timers   = {};  // key -> setTimeout id
  }

  subscribe(routerId, measurement, handler) {
    const key = `${routerId}:${measurement}`;
    if (!this._handlers[key]) this._handlers[key] = new Set();
    this._handlers[key].add(handler);
    if (!this._sources[key]) {
      this._retries[key] = 0;
      this._open(routerId, measurement, key);
    }
    return () => this.unsubscribe(routerId, measurement, handler);
  }

  unsubscribe(routerId, measurement, handler) {
    const key = `${routerId}:${measurement}`;
    this._handlers[key]?.delete(handler);
    if (!this._handlers[key]?.size) this._close(key);
  }

  closeAll() {
    Object.keys(this._sources).forEach(k => this._close(k));
  }

  _open(routerId, measurement, key) {
    const pathMap = {
      'hotspot/users':   `/routers/${routerId}/sse/hotspot/users`,
      'hotspot/active':  `/routers/${routerId}/sse/hotspot/active`,
      'hotspot/inactive':`/routers/${routerId}/sse/hotspot/inactive`,
      'ppp/secrets':     `/routers/${routerId}/sse/ppp/secrets`,
      'ppp/active':      `/routers/${routerId}/sse/ppp/active`,
      'ppp/inactive':    `/routers/${routerId}/sse/ppp/inactive`,
      'system/resource': `/routers/${routerId}/sse/system/resource`,
      'logs/all':        `/routers/${routerId}/logs/stream/all`,
      'logs/hotspot':    `/routers/${routerId}/logs/stream/hotspot`,
      'logs/ppp':        `/routers/${routerId}/logs/stream/ppp`,
    };

    const path = pathMap[measurement];
    if (!path) return;

    const url = `${SSE_BASE}${path}?token=${encodeURIComponent(Auth.token)}`;
    const es = new EventSource(url);
    this._sources[key] = es;

    es.onopen = () => {
      this._retries[key] = 0;
    };

    es.onmessage = (e) => {
      let data;
      try { data = JSON.parse(e.data); } catch { data = e.data; }
      this._handlers[key]?.forEach(fn => fn(data));
    };

    es.onerror = () => {
      // Don't reconnect if we deliberately closed (no handlers)
      if (!this._handlers[key]?.size) return;

      es.close();
      delete this._sources[key];

      // Exponential backoff: 2s, 4s, 8s … capped at 30s
      const retries = (this._retries[key] || 0) + 1;
      this._retries[key] = retries;
      const delay = Math.min(1000 * Math.pow(2, retries - 1), 30000);

      clearTimeout(this._timers[key]);
      this._timers[key] = setTimeout(() => {
        if (this._handlers[key]?.size) {
          this._open(routerId, measurement, key);
        }
      }, delay);
    };
  }

  _close(key) {
    clearTimeout(this._timers[key]);
    this._sources[key]?.close();
    delete this._sources[key];
    delete this._handlers[key];
    delete this._retries[key];
    delete this._timers[key];
  }
}

const SSE = new SSEManager();
