import { createSignal, createEffect, For } from 'solid-js';
import { del, getJSON, upload as uploadWithAuth } from '../api';
import { useToast } from '../components/Toast';
import Card from '../components/cards';

export function Drivers() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal('');

  const load = () => {
    setLoading(true);
    getJSON('/api/drivers')
      .then((res) => {
        setItems(res.data || res);
        setError('');
      })
      .catch(() => setError('加载驱动失败'))
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    load();
  });

  const remove = (id, name) => {
    if (!confirm(`删除驱动 ${name} ？`)) return;
    del(`/api/drivers/${id}`)
      .then(() => { toast.show('success', '已删除'); load(); })
      .catch(() => toast.show('error', '删除失败'));
  };

  const upload = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const fd = new FormData();
    fd.append('file', file);
    uploadWithAuth('/api/drivers/upload', fd)
      .then(() => { toast.show('success', '上传成功'); load(); })
      .catch(() => toast.show('error', '上传失败'))
      .finally(() => { e.target.value = ''; });
  };

  return (
    <Card
      title="驱动管理"
      extra={
        <label class="btn btn-primary" style="cursor:pointer;">
          上传驱动
          <input type="file" accept=".wasm" style="display:none" onChange={upload} />
        </label>
      }
    >
        {loading() ? (
          <div class="text-center" style="padding:48px; color:var(--text-muted);">
            <div class="loading-spinner" style="margin:0 auto 16px;"></div>
            <div>加载中...</div>
          </div>
        ) : error() ? (
          <div style="color:var(--accent-red); padding:16px 0;">{error()}</div>
        ) : (
          <div class="table-container" style="max-height:520px; overflow:auto;">
            <table class="table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>名称</th>
                  <th>文件</th>
                  <th>大小</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <For each={items()}>
                  {(d) => (
                    <tr>
                      <td>{d.id}</td>
                      <td>{d.name}</td>
                      <td>{d.filename || d.file_path || ''}</td>
                      <td>{d.size ? (d.size / 1024).toFixed(1) + ' KB' : '-'}</td>
                      <td class="flex" style="gap:8px;">
                        <a class="btn" href={`/api/drivers/${d.id}/download`}>下载</a>
                        <button class="btn btn-danger" onClick={() => remove(d.id, d.name)}>删除</button>
                      </td>
                    </tr>
                  )}
                </For>
                <For each={items().length === 0 ? [1] : []}>
                  {() => (
                    <tr>
                      <td colSpan={5} style="text-align:center; padding:24px; color:var(--text-muted);">暂无驱动</td>
                    </tr>
                  )}
                </For>
              </tbody>
            </table>
          </div>
        )}
    </Card>
  );
}
