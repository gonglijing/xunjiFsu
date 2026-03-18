import { Show } from 'solid-js';

export default function LoadErrorHint(props) {
  return (
    <Show when={props.error}>
      <div class="load-error-hint">
        <span class="inline-error">{props.error}</span>
        <button class="btn btn-ghost btn-sm" type="button" onClick={() => props.onRetry?.()}>
          重试
        </button>
      </div>
    </Show>
  );
}
