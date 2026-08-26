// useZones(pollMs) — polling hook over GET /api/overview.
// Shared by the overview page and the zone detail page. Returns a handle
// with start()/stop()/subscribe()/refresh() so pages can tear polling down
// when they unmount.

import { api } from '/api.js';

export function useZones(pollMs = 5000) {
  const store = { data: null, error: null, loading: true, updatedAt: null };
  const subs = new Set();
  let timer = null;
  let stopped = false;

  function notify() {
    subs.forEach((fn) => {
      try {
        fn(store);
      } catch (e) {
        console.error('[useZones] subscriber error', e);
      }
    });
  }

  async function refresh() {
    if (stopped) return;
    try {
      const data = await api('/api/overview');
      store.data = data;
      store.error = null;
      store.updatedAt = new Date();
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
      stopped = false;
      refresh();
      if (pollMs > 0) {
        timer = setInterval(refresh, pollMs);
      }
    },
    stop() {
      stopped = true;
      if (timer) clearInterval(timer);
      timer = null;
    },
    refresh,
  };
}
