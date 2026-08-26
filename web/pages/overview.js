// overview.js — clean zone dashboard (GET /). Uses ZoneCard + useZones +
// useAlerts.

import { useZones } from '/hooks/use_zones.js';
import { useAlerts } from '/hooks/use_alerts.js';
import { ZoneCard, escapeHtml } from '/components/zone_card.js';
import { fmtTime, fmtNum } from '/api.js';

export function render(container) {
  const zones = useZones(4000);
  const alerts = useAlerts({ limit: 8 }, 8000);
  zones.start();
  alerts.start();

  const unsubZones = zones.subscribe((s) => {
    renderOverview(container, s, alerts.state);
  });
  const unsubAlerts = alerts.subscribe((s) => {
    renderOverview(container, zones.state, s);
  });

  return () => {
    unsubZones();
    unsubAlerts();
    zones.stop();
    alerts.stop();
  };
}

function renderOverview(container, zoneState, alertState) {
  if (zoneState.loading && !zoneState.data) {
    container.innerHTML = '<div class="page-loading">加载总览…</div>';
    return;
  }
  if (zoneState.error && !zoneState.data) {
    container.innerHTML = `<div class="error-state">加载失败：${escapeHtml(zoneState.error)}</div>`;
    return;
  }
  const data = zoneState.data || { zones: [] };
  const zones = data.zones || [];
  const openAlerts = (alertState.data || []).filter((a) => a.status !== 'closed');

  container.innerHTML = `
    <div class="page-head">
      <div>
        <h2>洁净区总览</h2>
        <p class="muted">各分区等级状态 · 实时粒子值 · 未确认报警（${fmtTime(zoneState.updatedAt?.toISOString())} 更新）</p>
      </div>
      <div class="stat-strip">
        <div class="stat"><span class="stat-num">${data.total_zones ?? zones.length}</span><span class="stat-label">洁净区</span></div>
        <div class="stat"><span class="stat-num">${data.total_monitors ?? 0}</span><span class="stat-label">监测点</span></div>
        <div class="stat stat-alert"><span class="stat-num">${data.active_alerts ?? 0}</span><span class="stat-label">未处置报警</span></div>
        <div class="stat stat-interlock"><span class="stat-num">${data.open_interlocks ?? 0}</span><span class="stat-label">进行中联锁</span></div>
      </div>
    </div>
    <section class="alert-ticker">
      <h3 class="section-title">最新报警</h3>
      ${
        openAlerts.length
          ? `<ul class="ticker-list">
               ${openAlerts
                 .slice(0, 5)
                 .map(
                   (a) => `<li>
                     <span class="ticker-type type-${a.type}">${escapeHtml(a.type)}</span>
                     <span class="ticker-msg">${escapeHtml(a.message)}</span>
                     <span class="ticker-time">${fmtTime(a.created_at)}</span>
                     <span class="ticker-status">${escapeHtml(a.status)}</span>
                     <a class="ticker-link" href="/alerts" data-link>去处置 →</a>
                   </li>`,
                 )
                 .join('')}
             </ul>`
          : '<div class="empty-state">当前无未处置报警 ✓</div>'
      }
    </section>
    <section>
      <h3 class="section-title">洁净区卡片</h3>
      <div class="zone-grid" id="zone-grid"></div>
    </section>`;

  const grid = container.querySelector('#zone-grid');
  for (const z of zones) {
    grid.appendChild(ZoneCard(z));
  }
  if (!zones.length) {
    grid.innerHTML = '<div class="empty-state">暂无洁净区</div>';
  }
}
