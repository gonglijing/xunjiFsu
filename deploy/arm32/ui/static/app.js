// Global UI helpers for GoGW
(() => {
  const icons = { success: '✓', error: '✕', warning: '⚠', info: 'ℹ' };

  window.showToast = function(type, title, message, duration = 4000) {
    const container = document.getElementById('toast-container');
    if (!container) return;
    const toast = document.createElement('div');
    toast.className = `toast ${type || 'info'}`;
    toast.innerHTML = `
      <div class="toast-icon">${icons[type] || icons.info}</div>
      <div class="toast-content">
        <div class="toast-title">${title || ''}</div>
        ${message ? `<div class="toast-message">${message}</div>` : ''}
      </div>
      <button class="modal-close" aria-label="close" onclick="this.parentElement.remove()">&times;</button>
    `;
    container.appendChild(toast);
    if (duration > 0) {
      setTimeout(() => {
        toast.style.animation = 'toastSlideIn 0.25s ease-out reverse';
        setTimeout(() => toast.remove(), 250);
      }, duration);
    }
  };

  // Modal controls
  window.openModal = function(id) {
    closeAllModals();
    const modal = document.getElementById(id);
    const container = document.getElementById('modal-container');
    if (modal && container) {
      modal.classList.remove('hidden');
      container.classList.add('active');
    }
    if (id === 'device-modal') maybeLoadDrivers();
  };

  window.closeModal = function(id) {
    if (id) {
      const m = document.getElementById(id);
      if (m) m.classList.add('hidden');
    }
    const container = document.getElementById('modal-container');
    if (container) container.classList.remove('active');
  };

  window.closeAllModals = function() {
    document.querySelectorAll('.modal-content > div').forEach((el) => el.classList.add('hidden'));
    const container = document.getElementById('modal-container');
    if (container) container.classList.remove('active');
  };

  // Tabs
  window.switchTab = function(name) {
    document.querySelectorAll('.tab-content').forEach((el) => el.classList.remove('active'));
    document.querySelectorAll('.tab-btn').forEach((el) => el.classList.remove('active'));
    const content = document.getElementById(`tab-${name}`);
    const btn = document.getElementById(`tab-btn-${name}`);
    if (content) content.classList.add('active');
    if (btn) btn.classList.add('active');
  };

  // Driver upload
  window.uploadDriver = function(input) {
    if (!input.files || !input.files[0]) return;
    const formData = new FormData();
    formData.append('file', input.files[0]);

    fetch('/api/drivers/upload', { method: 'POST', body: formData })
      .then((res) => res.json())
      .then((data) => {
        htmx.ajax('GET', '/api/drivers', { target: '#tab-drivers', swap: 'innerHTML' });
        showToast(data.success ? 'success' : 'error', data.success ? '上传成功' : '上传失败', data.error || (data.data && data.data.filename));
      })
      .catch((err) => showToast('error', '上传失败', err.message))
      .finally(() => { input.value = ''; });
  };

  // Device driver list lazy load
  function maybeLoadDrivers() {
    const select = document.querySelector('#device-modal select[name="driver_id"]');
    if (!select || select.options.length > 1) return;
    fetch('/api/drivers')
      .then((res) => res.json())
      .then((data) => {
        if (!data.data || !Array.isArray(data.data)) return;
        select.innerHTML = '<option value="">选择驱动...</option>';
        data.data.forEach((driver, idx) => {
          const opt = document.createElement('option');
          opt.value = driver.id || idx + 1;
          opt.textContent = driver.name;
          select.appendChild(opt);
        });
      })
      .catch(() => {});
  }

  // Driver type toggle
  window.toggleDriverFields = function() {
    const driverType = document.getElementById('device-driver-type');
    const rtuFields = document.getElementById('modbus-rtu-fields');
    const tcpFields = document.getElementById('modbus-tcp-fields');
    if (!driverType || !rtuFields || !tcpFields) return;
    if (driverType.value === 'modbus_rtu') {
      rtuFields.classList.remove('hidden');
      tcpFields.classList.add('hidden');
    } else {
      rtuFields.classList.add('hidden');
      tcpFields.classList.remove('hidden');
    }
  };

  // Nav toggle (mobile)
  document.addEventListener('DOMContentLoaded', () => {
    const navToggle = document.querySelector('.nav-toggle');
    const navLinks = document.querySelector('.nav-links');
    if (navToggle && navLinks) {
      navToggle.addEventListener('click', () => navLinks.classList.toggle('active'));
    }

    const modal = document.getElementById('modal-container');
    if (modal) {
      modal.addEventListener('click', (e) => {
        const backdrop = modal.querySelector('.modal-backdrop');
        if (e.target === backdrop) closeAllModals();
      });
    }
  });

  // HTMX feedback hooks
  document.addEventListener('htmx:afterRequest', (evt) => {
    if (evt.detail.successful && evt.detail.xhr.status === 200) {
      const resp = evt.detail.xhr.responseText || '';
      if (resp.includes('success') || resp.includes('创建成功') || resp.includes('添加成功')) {
        showToast('success', '操作成功', '数据已更新');
      }
    }
  });

  document.addEventListener('htmx:responseError', () => {
    showToast('error', '请求失败', '请稍后重试');
  });
})();
