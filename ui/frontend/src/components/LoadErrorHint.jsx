import { Show } from 'solid-js';

export default function LoadErrorHint(props) {
  return (
    <Show when={props.error}>
      <div
        style="
          display:flex;
          align-items:center;
          justify-content:space-between;
          gap:12px;
          margin-bottom:12px;
          padding:10px 12px;
          border:1px solid rgba(239,68,68,0.35);
          border-radius:10px;
          background:rgba(239,68,68,0.08);
        "
      >
        <span style="color:var(--accent-red);">{props.error}</span>
        <button class="btn btn-ghost btn-sm" type="button" onClick={() => props.onRetry?.()}>
          重试
        </button>
      </div>
    </Show>
  );
}
