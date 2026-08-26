// useAlerts(filter) — alert listing hook over GET /api/alerts.
// Shared by the alert console and the overview page (which shows the most
// recent unacknowledged alerts).

import { api } from '/api.js';

export function useAlerts(filter = {}, pollMs = 0) {
  const store = { data: [], error: null, loading: true };
  const subs = new Set();
  let timer = null;

  function notify() {
    subs.forEach((fn) => {
      try {
        fn(store);
      } catch (e) {
        console.error('[useAlerts] subscriber error', e);
      }
    });
  }

  async function refresh() {
    try {
      const params = new URLSearchParams();
      if (filter.status) params.set('status', filter.status);
      if (filter.type) params.set('type', filter.type);
      if (filter.monitorZoneId) params.set('monitor_zone_id', filter.monitorZoneId);
      if (filter.limit) params.set('limit', String(filter.limit));
      const qs = params.toString();
      const data = await api('/api/alerts' + (qs ? '?' + qs : ''));
      store.data = data || [];
      store.error = null;
    } catch (e) {
      store.error = e.message;
    } finally {
      store.loading = false;
      notify();
    }
  }

  return {
    get state() {
      return store;
    },
    subscribe(fn) {
      subs.add(fn);
      fn(store);
      return () => subs.delete(fn);
    },
    start() {
      refresh();
      if (pollMs > 0) timer = setInterval(refresh, pollMs);
    },
    stop() {
      if (timer) clearInterval(timer);
      timer = null;
    },
    refresh,
  };
}
