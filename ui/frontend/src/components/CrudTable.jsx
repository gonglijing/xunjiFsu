import { For, Show } from 'solid-js';

function CrudTable(props) {
  const columns = () => props.columns || [];
  const rows = () => props.items || [];
  const loading = () => !!props.loading;

  const hasActions = () => typeof props.renderActions === 'function';
  const emptyText = () => props.emptyText || '暂无数据';

  return (
    <div class="table-container" style={props.style || ''}>
      <table class="table">
        <thead>
          <tr>
            <For each={columns()}>
              {(col) => <th>{col.title}</th>}
            </For>
            <Show when={hasActions()}>
              <th>操作</th>
            </Show>
          </tr>
        </thead>
        <tbody>
          <Show
            when={!loading()}
            fallback={
              <tr>
                <td colSpan={columns().length + (hasActions() ? 1 : 0)} style="text-align:center; padding:32px;">
                  <div class="loading-spinner" style="margin:0 auto 12px;"></div>
                  <div class="text-muted">加载中...</div>
                </td>
              </tr>
            }
          >
            <Show
              when={rows().length > 0}
              fallback={
                <tr>
                  <td colSpan={columns().length + (hasActions() ? 1 : 0)} style="text-align:center; padding:24px; color:var(--text-muted);">
                    {emptyText()}
                  </td>
                </tr>
              }
            >
              <For each={rows()}>
                {(row) => (
                  <tr>
                    <For each={columns()}>
                      {(col) => (
                        <td>
                          {col.render ? col.render(row) : row[col.key]}
                        </td>
                      )}
                    </For>
                    <Show when={hasActions()}>
                      <td>
                        {props.renderActions(row)}
                      </td>
                    </Show>
                  </tr>
                )}
              </For>
            </Show>
          </Show>
        </tbody>
      </table>
    </div>
  );
}

export default CrudTable;

