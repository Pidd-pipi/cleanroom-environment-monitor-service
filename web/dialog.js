// dialog.js — in-page input / confirm / toast dialogs.
// Some embedded browsers do not support window.prompt()/alert(), so every
// interaction that needs text input or confirmation renders its own modal
// instead. All functions return Promises.

export function askInput({ title = '请输入', placeholder = '', okLabel = '确定' } = {}) {
  return new Promise((resolve) => {
    const overlay = document.createElement('div');
    overlay.className = 'modal';
    overlay.innerHTML = `
      <div class="modal-box dialog-box">
        <div class="modal-head"><h3>${escapeHtml(title)}</h3></div>
        <input class="dialog-input" type="text" placeholder="${escapeHtml(placeholder)}" autocomplete="off" />
        <div class="modal-actions">
          <button class="btn" data-dlg-cancel>取消</button>
          <button class="btn btn-primary" data-dlg-ok>${escapeHtml(okLabel)}</button>
        </div>
      </div>`;
    document.body.appendChild(overlay);
    const input = overlay.querySelector('.dialog-input');
    const close = (value) => {
      overlay.remove();
      resolve(value);
    };
    const ok = () => close(input.value.trim());
    overlay.querySelector('[data-dlg-ok]').addEventListener('click', ok);
    overlay.querySelector('[data-dlg-cancel]').addEventListener('click', () => close(null));
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) close(null);
    });
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') ok();
      if (e.key === 'Escape') close(null);
    });
    setTimeout(() => input.focus(), 0);
  });
}

export function askConfirm({ title = '请确认', okLabel = '确定', danger = false } = {}) {
  return new Promise((resolve) => {
    const overlay = document.createElement('div');
    overlay.className = 'modal';
    overlay.innerHTML = `
      <div class="modal-box dialog-box">
        <div class="modal-head"><h3>${escapeHtml(title)}</h3></div>
        <div class="modal-actions">
          <button class="btn" data-dlg-cancel>取消</button>
          <button class="btn ${danger ? 'btn-danger' : 'btn-primary'}" data-dlg-ok>${escapeHtml(okLabel)}</button>
        </div>
      </div>`;
    document.body.appendChild(overlay);
    const close = (v) => {
      overlay.remove();
      resolve(v);
    };
    overlay.querySelector('[data-dlg-ok]').addEventListener('click', () => close(true));
    overlay.querySelector('[data-dlg-cancel]').addEventListener('click', () => close(false));
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) close(false);
    });
  });
}

export function notify(message, kind = 'info') {
  const el = document.createElement('div');
  el.className = 'toast toast-' + kind;
  el.textContent = message;
  document.body.appendChild(el);
  setTimeout(() => el.classList.add('toast-show'), 20);
  setTimeout(() => {
    el.classList.remove('toast-show');
    setTimeout(() => el.remove(), 300);
  }, 3200);
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
