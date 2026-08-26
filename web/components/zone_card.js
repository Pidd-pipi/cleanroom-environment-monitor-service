// ZoneCard — reusable clean-zone status card.
// Shared by: overview page (grid) and zone detail page (current status).

import { fmtNum, pct, fmtTime } from '/api.js';

export const STATUS_LABEL = {
  normal: '正常运行',
  elevated: '粒子偏高',
  over_limit: '超限',
  interlocked: '联锁通风',
  restored: '恢复确认',
};

export const STATUS_CLASS = {
  normal: 'status-normal',
  elevated: 'status-elevated',
  over_limit: 'status-over',
  interlocked: 'status-interlocked',
  restored: 'status-restored',
};

export const ISO_LABEL = {
  iso5: 'ISO 5',
  iso6: 'ISO 6',
  iso7: 'ISO 7',
  iso8: 'ISO 8',
};

export const PROCESS_LABEL = {
  lithography: '光刻',
  etching: '刻蚀',
  diffusion: '扩散',
};

export function ZoneCard(zone, { onSelect } = {}) {
  const cz = zone.clean_zone || {};
  const monitors = zone.monitor_zones || [];
  const latest = monitors.map((m) => m.latest_sample).filter(Boolean).sort((a, b) =>
    new Date(b.timestamp) - new Date(a.timestamp))[0];
  const status = cz.status || 'normal';
  const card = document.createElement('article');
  card.className = 'zone-card ' + (zone.interlocked ? 'zone-card-interlocked' : '');
  card.tabIndex = 0;

  const ratio = cz.last_particle_ratio ?? (latest ? latest.ratio : null);
  const ratioHtml = ratio !== null && ratio !== undefined
    ? `<div class="metric"><span class="metric-label">粒子比</span><span class="metric-value">${fmtNum(ratio, 2)}×</span></div>`
    : '';

  card.innerHTML = `
    <div class="zone-card-head">
      <div>
        <h3 class="zone-name">${escapeHtml(cz.name || cz.id || '-')}</h3>
        <div class="zone-sub">${escapeHtml(cz.physical_area || '')} · ${PROCESS_LABEL[cz.process] || cz.process || '-'}</div>
      </div>
      <span class="status-badge ${STATUS_CLASS[status] || ''}">${STATUS_LABEL[status] || status}</span>
    </div>
    <div class="zone-card-body">
      <div class="metrics">
        <div class="metric"><span class="metric-label">ISO</span><span class="metric-value">${ISO_LABEL[cz.iso_class] || cz.iso_class || '-'}</span></div>
        <div class="metric"><span class="metric-label">监测点</span><span class="metric-value">${monitors.length}</span></div>
        <div class="metric"><span class="metric-label">未确认报警</span><span class="metric-value ${zone.active_alerts ? 'metric-alert' : ''}">${zone.active_alerts || 0}</span></div>
        ${ratioHtml}
      </div>
      ${
        latest
          ? `<div class="latest-sample">
              <span>0.3μm <b>${fmtNum(latest.count_0_3um)}</b></span>
              <span>0.5μm <b>${fmtNum(latest.count_0_5um)}</b></span>
              <span>温度 <b>${fmtNum(latest.temperature, 1)}℃</b></span>
              <span>湿度 <b>${fmtNum(latest.humidity, 1)}%</b></span>
              <span>压差 <b>${fmtNum(latest.pressure_diff, 1)}Pa</b></span>
            </div>`
          : '<div class="latest-sample muted">暂无采样数据</div>'
      }
      <div class="zone-card-foot">
        <span class="muted">状态自 ${fmtTime(cz.status_since)}</span>
        <span class="invalid-ratio">无效占比 ${pct(monitors[0]?.invalid_ratio ?? 0)}</span>
      </div>
    </div>`;

  card.addEventListener('click', () => {
    if (onSelect) onSelect(cz.id);
    else window.location.href = '/zones/' + encodeURIComponent(cz.id);
  });
  card.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') card.click();
  });
  return card;
}

export function escapeHtml(s) {
  if (s === null || s === undefined) return '';
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
