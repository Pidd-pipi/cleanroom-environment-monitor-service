// InterlockTimeline — reusable vertical timeline of interlock events.
// Shared by: interlocks page and zone detail page.

import { fmtTime } from '/api.js';

export const ACTION_LABEL = {
  ffu_speed_up: 'FFU 提速',
  fresh_air_increase: '新风加大',
  exhaust_increase: '排风增强',
};

export function InterlockTimeline(logs, { title = '联锁记录' } = {}) {
  const wrap = document.createElement('div');
  wrap.className = 'timeline-wrap';
  wrap.innerHTML = `<h3 class="section-title">${title}</h3>`;

  if (!logs || logs.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'empty-state';
    empty.textContent = '暂无联锁记录';
    wrap.appendChild(empty);
    return wrap;
  }

  const list = document.createElement('ul');
  list.className = 'timeline';
  for (const log of logs) {
    const li = document.createElement('li');
    li.className = 'timeline-item' + (log.restored_at ? '' : ' timeline-item-open');
    li.innerHTML = `
      <div class="timeline-dot"></div>
      <div class="timeline-content">
        <div class="timeline-head">
          <span class="timeline-action">${ACTION_LABEL[log.action] || log.action} · L${log.level}</span>
          <span class="timeline-time">${fmtTime(log.issued_at)}</span>
        </div>
        <div class="timeline-detail">
          <span>物理区 <b>${escapeHtml(log.physical_area || '-')}</b></span>
          <span>触发点 <b>${escapeHtml(log.trigger_monitor_zone_id || '-')}</b></span>
          <span>峰值比 <b>${Number(log.peak_ratio || 0).toFixed(2)}×</b></span>
        </div>
        <div class="timeline-detail muted">
          <span>原因 <b>${escapeHtml(log.reason || '-')}</b></span>
          <span>影响分区 <b>${(log.affected_zone_ids || []).join(', ') || '-'}</b></span>
        </div>
        ${
          log.restored_at
            ? `<div class="timeline-restore">
                已恢复 ${fmtTime(log.restored_at)} · ${escapeHtml(log.restore_by || '-')} · ${escapeHtml(log.restore_note || '')}
               </div>`
            : '<div class="timeline-restore open">联锁进行中…</div>'
        }
      </div>`;
    list.appendChild(li);
  }
  wrap.appendChild(list);
  return wrap;
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
