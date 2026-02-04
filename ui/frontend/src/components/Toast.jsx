import { createSignal, onMount, onCleanup } from 'solid-js';

function Toast(props) {
  const [toasts, setToasts] = createSignal([]);

  const show = (type, message) => {
    const id = Date.now();
    setToasts((prev) => [...prev, { id, type, message }]);
    setTimeout(() => {
      remove(id);
    }, 3000);
  };

  const remove = (id) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  // 暴露给全局
  if (typeof window !== 'undefined') {
    window.toast = { show };
  }

  return (
    <div class="toast-container" style="position:fixed; top:16px; right:16px; z-index:9999;">
      {toasts().map((t) => (
        <div
          key={t.id}
          class={`toast toast-${t.type}`}
          style={{
            "background": t.type === 'success' ? 'var(--accent-green)' : 
                        t.type === 'error' ? 'var(--accent-red)' : 
                        'var(--accent-blue)',
            "color": '#fff',
            "padding": '12px 20px',
            "border-radius": '4px',
            "margin-bottom": '8px',
            "box-shadow": '0 2px 8px rgba(0,0,0,0.15)',
            "animation": 'slideIn 0.3s ease'
          }}
        >
          {t.message}
          <button
            style={{
              "background": "none",
              "border": "none",
              "color": "#fff",
              "margin-left": "12px",
              "cursor": "pointer"
            }}
            onClick={() => remove(t.id)}
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
}

export function useToast() {
  return {
    show: (type, message) => {
      if (typeof window !== 'undefined' && window.toast) {
        window.toast.show(type, message);
      }
    }
  };
}

export default Toast;
