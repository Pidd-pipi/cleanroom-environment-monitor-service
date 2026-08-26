// alerts.js — alert console (/alerts). Uses useAlerts + ParticleTrend.

import { useAlerts } from '/hooks/use_alerts.js';
import { api, post, fmtTime, fmtNum } from '/api.js';
import { escapeHtml } from '/components/zone_card.js';
import { ParticleTrend } from '/components/particle_trend.js';
import { askInput, notify } from '/dialog.js';

const TYPE_LABEL = {
  particle: '粒子',
  temp_humidity: '温湿度',
  pressure: '压差',
  data_quality: '数据可信度',
};

export function render(container) {
  const state = { status: '', type: '', alerts: null };
  const hook = useAlerts({}, 5000);
  hook.start();

  const unsub = hook.subscribe((s) => {
    state.alerts = s;
    renderTable(container, state, hook);
  });

  return () => {
    unsub();
    hook.stop();
  };
}

function renderTable(container, state, hook) {
  const s = state.alerts;
  if (!s || (s.loading && !s.data.length)) {
    container.innerHTML = '<div class="page-loading">加载报警台…</div>';
    return;
  }
  container.innerHTML = `
    <div class="page-head">
      <div><h2>报警台</h2><p class="muted">确认处置 · 升级 · 去重（20 分钟内同类合并）</p></div>
      <div class="filter-row">
        <select id="filter-status">
          <option value="">全部状态</option>
          <option value="open" ${state.status === 'open' ? 'selected' : ''}>未确认</option>
          <option value="acknowledged" ${state.status === 'acknowledged' ? 'selected' : ''}>已确认</option>
          <option value="escalated" ${state.status === 'escalated' ? 'selected' : ''}>已升级</option>
          <option value="closed" ${state.status === 'closed' ? 'selected' : ''}>已关闭</option>
        </select>
        <select id="filter-type">
          <option value="">全部类型</option>
          ${Object.entries(TYPE_LABEL)
            .map(
              ([k, v]) =>
                `<option value="${k}" ${state.type === k ? 'selected' : ''}>${v}</option>`,
            )
            .join('')}
        </select>
        <button class="btn" id="btn-refresh">刷新</button>
      </div>
    </div>
    <section class="card">
      <table class="data-table">
        <thead><tr>
          <th>级别</th><th>类型</th><th>分区/监测点</th><th>消息</th>
          <th>次数</th><th>状态</th><th>首次出现</th><th>操作</th>
        </tr></thead>
        <tbody>
          ${s.data.length
            ? s.data
                .map(
                  (a) => `<tr class="alert-row alert-${a.level}">
                    <td><span class="level-badge level-${a.level}">${escapeHtml(a.level)}</span></td>
                    <td><span class="ticker-type type-${a.type}">${TYPE_LABEL[a.type] || escapeHtml(a.type)}</span></td>
                    <td>${escapeHtml(a.monitor_zone_id)}</td>
                    <td class="alert-msg">${escapeHtml(a.message)}</td>
                    <td>${a.count || 1}</td>
                    <td><span class="alert-status status-${a.status}">${escapeHtml(a.status)}</span></td>
                    <td>${fmtTime(a.first_seen_at)}</td>
                    <td class="action-cell">
                      <button class="btn btn-small" data-trend="${escapeHtml(a.id)}">趋势</button>
                      ${
                        a.status !== 'closed' && a.status !== 'acknowledged'
                          ? `<button class="btn btn-small btn-primary" data-ack="${escapeHtml(a.id)}">确认</button>`
                          : ''
                      }
                      ${
                        a.status !== 'closed' && a.status !== 'escalated'
                          ? `<button class="btn btn-small btn-warn" data-esc="${escapeHtml(a.id)}">升级</button>`
                          : ''
                      }
                    </td>
                  </tr>`,
                )
                .join('')
            : '<tr><td colspan="8" class="empty-state">暂无报警</td></tr>'}
        </tbody>
      </table>
    </section>
    <div id="trend-modal" class="modal hidden">
      <div class="modal-box">
        <div class="modal-head"><h3>粒子趋势</h3><button class="btn btn-small" id="modal-close">关闭</button></div>
        <div id="modal-body"></div>
      </div>
    </div>`;

  container.querySelector('#filter-status').addEventListener('change', (e) => {
    state.status = e.target.value;
    hook.refresh();
  });
  container.querySelector('#filter-type').addEventListener('change', (e) => {
    state.type = e.target.value;
    hook.refresh();
  });
  container.querySelector('#btn-refresh').addEventListener('click', () => hook.refresh());

  container.querySelectorAll('[data-ack]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      const id = btn.dataset.ack;
      const disposition = await askInput({ title: '请输入处置说明（必填）', placeholder: '例如：擦拭传感器镜片并复测' });
      if (disposition === null) return;
      if (!disposition.trim()) { notify('处置说明不能为空', 'error'); return; }
      try {
        await post('/api/alerts/' + encodeURIComponent(id) + '/ack', {
          operator: 'web_user',
          disposition: disposition.trim(),
        });
        hook.refresh();
        notify('报警确认成功', 'success');
      } catch (e) {
        notify('确认失败：' + e.message, 'error');
      }
    });
  });

  container.querySelectorAll('[data-esc]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      const id = btn.dataset.esc;
      try {
        await post('/api/alerts/' + encodeURIComponent(id) + '/escalate', { operator: 'web_user' });
        hook.refresh();
      } catch (e) {
        notify('升级失败：' + e.message, 'error');
      }
    });
  });

  container.querySelectorAll('[data-trend]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      const alertRow = s.data.find((a) => a.id === btn.dataset.trend);
      if (!alertRow) return;
      const modal = container.querySelector('#trend-modal');
      const body = container.querySelector('#modal-body');
      body.innerHTML = '<div class="page-loading">加载趋势…</div>';
      modal.classList.remove('hidden');
      try {
        const samples = await api(
          '/api/monitors/' + encodeURIComponent(alertRow.monitor_zone_id) + '/samples?limit=60',
        );
        body.innerHTML = '';
        body.appendChild(ParticleTrend(samples || [], { title: alertRow.monitor_zone_id + ' 粒子趋势' }));
      } catch (e) {
        body.innerHTML = '<div class="error-state">' + escapeHtml(e.message) + '</div>';
      }
    });
  });

  const modal = container.querySelector('#trend-modal');
  modal.addEventListener('click', (e) => {
    if (e.target === modal || e.target.id === 'modal-close') modal.classList.add('hidden');
  });
}
