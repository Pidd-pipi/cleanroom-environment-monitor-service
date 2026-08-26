// zone_detail.js — clean zone detail page (/zones/{id}).
// Uses ZoneCard + ParticleTrend + InterlockTimeline + useZones.

import { useZones } from '/hooks/use_zones.js';
import { api, post, fmtNum, fmtTime } from '/api.js';
import { ZoneCard, escapeHtml, ISO_LABEL, PROCESS_LABEL, STATUS_LABEL } from '/components/zone_card.js';
import { ParticleTrend } from '/components/particle_trend.js';
import { askInput, askConfirm, notify } from '/dialog.js';
import { InterlockTimeline } from '/components/interlock_timeline.js';

export function render(container, params) {
  const zoneId = params.id;
  let zones;
  let unsub = () => {};
  const zonesHook = useZones(4000);
  zonesHook.start();
  unsub = zonesHook.subscribe((s) => {
    zones = s.data || { zones: [] };
    renderPage(container, zoneId, zones, () => zonesHook.refresh());
  });
  return () => {
    unsub();
    zonesHook.stop();
  };
}

async function renderPage(container, zoneId, overview, refresh) {
  const zone = (overview.zones || []).find((z) => z.clean_zone.id === zoneId);
  if (!zone) {
    container.innerHTML = '<div class="error-state">洁净区不存在或加载中…</div>';
    return;
  }
  let samples = [];
  let interlocks = [];
  try {
    const [s, il] = await Promise.all([
      api('/api/zones/' + encodeURIComponent(zoneId) + '/samples?limit=120'),
      api('/api/zones/' + encodeURIComponent(zoneId) + '/interlocks'),
    ]);
    samples = s || [];
    interlocks = il || [];
  } catch (e) {
    container.innerHTML = `<div class="error-state">数据加载失败：${escapeHtml(e.message)}</div>`;
    return;
  }

  const cz = zone.clean_zone;
  const isInterlocked = cz.status === 'interlocked';
  const isRestorable = isInterlocked;
  // The state machine only allows entering interlocked from
  // normal / elevated / over_limit; restored needs a good sample first.
  const canInterlock = ['normal', 'elevated', 'over_limit'].includes(cz.status);

  container.innerHTML = `
    <div class="page-head">
      <div>
        <h2><a href="/" data-link class="back-link">←</a> 分区详情</h2>
        <p class="muted">${escapeHtml(cz.name)} · ${escapeHtml(cz.physical_area)} · ${PROCESS_LABEL[cz.process] || cz.process}</p>
      </div>
      <div class="action-row">
        ${
          isRestorable
            ? `<button class="btn btn-primary" id="btn-restore">恢复确认</button>`
            : ''
        }
        <button class="btn btn-danger" id="btn-interlock" ${canInterlock ? '' : 'disabled'} title="${canInterlock ? '' : '当前状态不可联锁（需先恢复至正常）'}">手动联锁</button>
      </div>
    </div>

    <div class="detail-grid">
      <section class="card" id="zone-card-slot"></section>
      <section class="card">
        <h3 class="section-title">等级判定明细</h3>
        <div class="judge-table">
          <div class="judge-row"><span>设计等级</span><b>${ISO_LABEL[cz.iso_class] || cz.iso_class}</b></div>
          <div class="judge-row"><span>状态机</span><b>${STATUS_LABEL[cz.status] || cz.status}</b></div>
          <div class="judge-row"><span>当前粒子比</span><b>${fmtNum(cz.last_particle_ratio, 2)}×</b></div>
          <div class="judge-row"><span>最近判定等级</span><b id="judge-iso">-</b></div>
          <div class="judge-row"><span>监测点数</span><b>${zone.monitor_zones.length}</b></div>
        </div>
      </section>
    </div>

    <section class="card">
      <h3 class="section-title">粒子趋势（双粒径）</h3>
      <div id="trend-slot"></div>
    </section>

    <section class="card">
      <h3 class="section-title">监测点</h3>
      <div id="monitor-slot"></div>
    </section>

    <section class="card">
      <div id="timeline-slot"></div>
    </section>
    `;

  const cardSlot = container.querySelector('#zone-card-slot');
  cardSlot.appendChild(ZoneCard(zone));

  const latestValid = [...samples].reverse().find((s) => s.valid);
  const judgeEl = container.querySelector('#judge-iso');
  if (latestValid) {
    judgeEl.textContent = latestValid.iso_class ? latestValid.iso_class.toUpperCase() : '超出表范围';
  } else {
    judgeEl.textContent = '-';
  }

  container.querySelector('#trend-slot').appendChild(ParticleTrend(samples, { title: '近 ' + samples.length + ' 条采样' }));
  container.querySelector('#timeline-slot').appendChild(InterlockTimeline(interlocks, { title: '联锁记录' }));

  // Monitor zone table
  const monitorSlot = container.querySelector('#monitor-slot');
  const table = document.createElement('table');
  table.className = 'data-table';
  table.innerHTML = `
    <thead><tr>
      <th>监测点</th><th>粒子计数器</th><th>最新采样</th><th>FFU</th><th>新风</th><th>维护中</th><th>标定到期</th>
    </tr></thead>
    <tbody>
      ${zone.monitor_zones
        .map((m) => {
          const s = m.latest_sample;
          return `<tr>
            <td><a class="link" href="/equipment" data-link>${escapeHtml(m.monitor_zone.name)}</a></td>
            <td>${escapeHtml(m.monitor_zone.particle_counter_id)}</td>
            <td>${s ? fmtTime(s.timestamp) + ' · ' + fmtNum(s.count_0_3um) + '/m³' : '-'}</td>
            <td>${m.monitor_zone.equipment.ffu_level}%</td>
            <td>${m.monitor_zone.equipment.fresh_air_ratio}%</td>
            <td>${m.monitor_zone.equipment.in_maintenance ? '⚠ 是' : '否'}</td>
            <td>${fmtTime(m.monitor_zone.equipment.calibration_due)}</td>
          </tr>`;
        })
        .join('')}
    </tbody>`;
  monitorSlot.appendChild(table);

  const restoreBtn = container.querySelector('#btn-restore');
  if (restoreBtn) {
    restoreBtn.addEventListener('click', async () => {
      const confirmed = await askConfirm({ title: '确认恢复分区 ' + cz.name + '？', okLabel: '确认恢复', danger: true });
      if (!confirmed) return;
      const note = await askInput({ title: '恢复确认说明（必填）', placeholder: '例如：更换过滤器并复测' });
      if (note === null) return;
      if (!note.trim()) { notify('恢复确认说明不能为空', 'error'); return; }
      try {
        await post('/api/zones/' + encodeURIComponent(zoneId) + '/restore', {
          operator: 'web_user',
          note: note.trim(),
        });
        refresh();
        notify('恢复确认成功', 'success');
      } catch (e) {
        notify('恢复确认失败：' + e.message, 'error');
      }
    });
  }
  const ilBtn = container.querySelector('#btn-interlock');
  if (ilBtn) {
    ilBtn.addEventListener('click', async () => {
      try {
        await post('/api/zones/' + encodeURIComponent(zoneId) + '/interlock', {
          reason: 'manual_interlock_from_ui',
          ratio: 1.5,
        });
        refresh();
        notify('联锁已下发（整区一致）', 'success');
      } catch (e) {
        notify('联锁下发失败：' + e.message, 'error');
      }
    });
  }
}
