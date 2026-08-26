// equipment.js — equipment status page (/equipment). Shows particle
// counters / FFU state and lets operators toggle PM maintenance.

import { api, post, fmtTime } from '/api.js';
import { escapeHtml } from '/components/zone_card.js';
import { askInput, notify } from '/dialog.js';

export function render(container) {
  container.innerHTML = '<div class="page-loading">加载设备状态…</div>';
  let cancelled = false;

  async function load() {
    try {
      const overview = await api('/api/overview');
      if (cancelled) return;
      const monitors = (overview.zones || []).flatMap((z) =>
        (z.monitor_zones || []).map((m) => ({ ...m, clean_zone: z.clean_zone })),
      );
      container.innerHTML = `
        <div class="page-head">
          <div><h2>设备状态</h2><p class="muted">粒子计数器 / FFU 状态 · PM 维护标记（维护中数据标记为无效）</p></div>
          <button class="btn" id="btn-reload">刷新</button>
        </div>
        <section class="card">
          <table class="data-table">
            <thead><tr>
              <th>监测点</th><th>所属洁净区</th><th>粒子计数器</th>
              <th>FFU 等级</th><th>新风比</th><th>标定到期</th>
              <th>维护状态</th><th>操作</th>
            </tr></thead>
            <tbody>
              ${monitors
                .map(
                  (m) => `<tr class="${m.monitor_zone.equipment.in_maintenance ? 'row-maintenance' : ''}">
                    <td>${escapeHtml(m.monitor_zone.name)}</td>
                    <td>${escapeHtml(m.clean_zone.name)}</td>
                    <td>${escapeHtml(m.monitor_zone.particle_counter_id)}</td>
                    <td>${m.monitor_zone.equipment.ffu_level}%</td>
                    <td>${m.monitor_zone.equipment.fresh_air_ratio}%</td>
                    <td>${fmtTime(m.monitor_zone.equipment.calibration_due)}</td>
                    <td>
                      ${
                        m.monitor_zone.equipment.in_maintenance
                          ? `<span class="alert-status status-elevated">维护中</span><div class="muted">${escapeHtml(m.monitor_zone.equipment.maintenance_note || '')}</div>`
                          : '<span class="alert-status status-normal">正常</span>'
                      }
                    </td>
                    <td>
                      ${
                        m.monitor_zone.equipment.in_maintenance
                          ? `<button class="btn btn-small btn-primary" data-pm-end="${escapeHtml(m.monitor_zone.id)}">结束维护</button>`
                          : `<button class="btn btn-small btn-warn" data-pm-start="${escapeHtml(m.monitor_zone.id)}">PM 维护</button>`
                      }
                    </td>
                  </tr>`,
                )
                .join('')}
            </tbody>
          </table>
        </section>`;

      container.querySelector('#btn-reload').addEventListener('click', load);
      container.querySelectorAll('[data-pm-start]').forEach((btn) => {
        btn.addEventListener('click', async () => {
          const note = await askInput({ title: '维护说明（必填）', placeholder: '例如：PM 校准' });
          if (note === null) return;
          if (!note.trim()) { notify('维护说明不能为空', 'error'); return; }
          try {
            await post('/api/monitors/' + encodeURIComponent(btn.dataset.pmStart) + '/maintenance', {
              in_maintenance: true,
              note: note.trim(),
            });
            notify('已进入 PM 维护', 'success');
            load();
          } catch (e) {
            notify('操作失败：' + e.message, 'error');
          }
        });
      });
      container.querySelectorAll('[data-pm-end]').forEach((btn) => {
        btn.addEventListener('click', async () => {
          try {
            await post('/api/monitors/' + encodeURIComponent(btn.dataset.pmEnd) + '/maintenance', {
              in_maintenance: false,
              note: 'PM 完成',
            });
            notify('PM 维护结束', 'success');
            load();
          } catch (e) {
            notify('操作失败：' + e.message, 'error');
          }
        });
      });
    } catch (e) {
      if (!cancelled) {
        container.innerHTML = `<div class="error-state">加载失败：${escapeHtml(e.message)}</div>`;
      }
    }
  }

  load();
  return () => {
    cancelled = true;
  };
}
