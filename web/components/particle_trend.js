// ParticleTrend — reusable particle concentration trend chart (canvas).
// Shared by: zone detail page and alert console (particle alerts).

import { fmtNum, fmtTime } from '/api.js';

export function ParticleTrend(samples, { width = 560, height = 220, title = '粒子浓度趋势' } = {}) {
  const wrap = document.createElement('div');
  wrap.className = 'trend-wrap';
  const canvas = document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;
  canvas.className = 'trend-canvas';
  wrap.innerHTML = `<div class="trend-title">${title}</div>`;
  wrap.appendChild(canvas);

  const sorted = [...(samples || [])].sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
  const ctx = canvas.getContext('2d');
  draw(ctx, canvas, sorted);
  return wrap;
}

function draw(ctx, canvas, samples) {
  const pad = { top: 18, right: 12, bottom: 28, left: 58 };
  const w = canvas.width;
  const h = canvas.height;
  ctx.clearRect(0, 0, w, h);

  if (!samples.length) {
    ctx.fillStyle = '#94a3b8';
    ctx.font = '13px system-ui, sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText('暂无采样数据', w / 2, h / 2);
    return;
  }

  const series = [
    { key: 'count_0_3um', label: '≥0.3μm', color: '#2563eb' },
    { key: 'count_0_5um', label: '≥0.5μm', color: '#dc2626' },
  ];
  const allValues = samples.flatMap((s) => [Number(s.count_0_3um) || 0, Number(s.count_0_5um) || 0]);
  const max = Math.max(1, ...allValues);

  // Grid
  ctx.strokeStyle = '#e2e8f0';
  ctx.fillStyle = '#64748b';
  ctx.font = '11px system-ui, sans-serif';
  ctx.textAlign = 'right';
  const steps = 4;
  for (let i = 0; i <= steps; i++) {
    const y = pad.top + ((h - pad.top - pad.bottom) * i) / steps;
    const val = max - (max * i) / steps;
    ctx.beginPath();
    ctx.moveTo(pad.left, y);
    ctx.lineTo(w - pad.right, y);
    ctx.stroke();
    ctx.fillText(compactNum(val), pad.left - 6, y + 4);
  }

  // X labels
  ctx.textAlign = 'center';
  const labelEvery = Math.max(1, Math.floor(samples.length / 6));
  samples.forEach((s, i) => {
    if (i % labelEvery !== 0 && i !== samples.length - 1) return;
    const x = pad.left + ((w - pad.left - pad.right) * i) / Math.max(1, samples.length - 1);
    ctx.fillText(shortTime(s.timestamp), x, h - 8);
  });

  const xAt = (i) =>
    pad.left + ((w - pad.left - pad.right) * i) / Math.max(1, samples.length - 1);
  const yAt = (v) => pad.top + (h - pad.top - pad.bottom) * (1 - (Number(v) || 0) / max);

  // Legend
  ctx.textAlign = 'left';
  let lx = pad.left;
  for (const s of series) {
    ctx.fillStyle = s.color;
    ctx.fillRect(lx, 6, 12, 4);
    ctx.fillStyle = '#334155';
    ctx.fillText(s.label, lx + 16, 10);
    lx += 16 + ctx.measureText(s.label).width + 24;
  }

  for (const s of series) {
    ctx.strokeStyle = s.color;
    ctx.lineWidth = 2;
    ctx.beginPath();
    samples.forEach((sample, i) => {
      const x = xAt(i);
      const y = yAt(sample[s.key]);
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.stroke();
  }

  // Last point dots
  for (const s of series) {
    const last = samples[samples.length - 1];
    ctx.fillStyle = s.color;
    ctx.beginPath();
    ctx.arc(xAt(samples.length - 1), yAt(last[s.key]), 3.5, 0, Math.PI * 2);
    ctx.fill();
  }
}

function compactNum(v) {
  if (v >= 1e6) return (v / 1e6).toFixed(1) + 'M';
  if (v >= 1e3) return (v / 1e3).toFixed(0) + 'K';
  return fmtNum(v);
}

function shortTime(iso) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  return `${hh}:${mm}`;
}

export function SampleTable(samples) {
  const table = document.createElement('table');
  table.className = 'data-table';
  table.innerHTML = `
    <thead>
      <tr>
        <th>时间</th><th>≥0.3μm</th><th>≥0.5μm</th><th>温度</th>
        <th>湿度</th><th>压差</th><th>ISO</th><th>有效</th>
      </tr>
    </thead>
    <tbody>
      ${(samples || [])
        .map(
          (s) => `<tr class="${s.valid ? '' : 'row-invalid'}">
            <td>${fmtTime(s.timestamp)}</td>
            <td>${fmtNum(s.count_0_3um)}</td>
            <td>${fmtNum(s.count_0_5um)}</td>
            <td>${fmtNum(s.temperature, 1)}</td>
            <td>${fmtNum(s.humidity, 1)}%</td>
            <td>${fmtNum(s.pressure_diff, 1)} Pa</td>
            <td>${s.iso_class ? s.iso_class.toUpperCase() : '-'}</td>
            <td>${s.valid ? '✓' : '✗ ' + (s.invalid_reason || '')}</td>
          </tr>`,
        )
        .join('')}
    </tbody>`;
  return table;
}
