// interlocks.js — interlock log page (/interlocks). Uses
// InterlockTimeline.

import { api } from '/api.js';
import { InterlockTimeline } from '/components/interlock_timeline.js';

export function render(container) {
  container.innerHTML = '<div class="page-loading">加载联锁记录…</div>';
  let cancelled = false;

  (async () => {
    try {
      const [logs, overview] = await Promise.all([
        api('/api/interlocks?limit=200'),
        api('/api/overview'),
      ]);
      if (cancelled) return;
      const open = (logs || []).filter((l) => !l.restored_at).length;
      container.innerHTML = `
        <div class="page-head">
          <div><h2>联锁记录</h2><p class="muted">联锁动作时间线 · 整区一致</p></div>
          <div class="stat-strip">
            <div class="stat stat-interlock"><span class="stat-num">${open}</span><span class="stat-label">进行中</span></div>
            <div class="stat"><span class="stat-num">${(logs || []).length}</span><span class="stat-label">累计</span></div>
            <div class="stat"><span class="stat-num">${overview.interlocked_zones || 0}</span><span class="stat-label">联锁中分区</span></div>
          </div>
        </div>
        <div id="timeline-slot"></div>`;
      const slot = container.querySelector('#timeline-slot');
      slot.appendChild(InterlockTimeline(logs || [], { title: '全部联锁事件' }));
    } catch (e) {
      if (!cancelled) {
        container.innerHTML = `<div class="error-state">加载失败：${escapeHtml(e.message)}</div>`;
      }
    }
  })();

  return () => {
    cancelled = true;
  };
}

function escapeHtml(s) {
  if (s === null || s === undefined) return '';
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
