import { createSignal, onMount } from 'solid-js';
import api from '../api/services';
import { useToast } from '../components/Toast';
import Card from '../components/cards';
import CrudTable from '../components/CrudTable';
import { showErrorToast, withErrorToast } from '../utils/errors';
import { usePageLoader } from '../utils/pageLoader';
import LoadErrorHint from '../components/LoadErrorHint';

export function Drivers() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [busyId, setBusyId] = createSignal(0);
  const {
    loading,
    error,
    setError,
    run: runDriversLoad,
  } = usePageLoader(async () => {
    const res = await api.drivers.listDrivers();
    setItems(res || []);
  }, {
    errorMessage: '加载驱动失败',
  });
  const showUploadError = withErrorToast(toast, '上传失败');
  const showReloadError = withErrorToast(toast, '重载失败');

  const load = () => {
    setError('');
    runDriversLoad();
  };

  onMount(load);

  const remove = (id, name) => {
    if (!confirm(`删除驱动 ${name} ？`)) return;
    api.drivers.deleteDriver(id)
      .then(() => { toast.show('success', '已删除'); load(); })
      .catch((err) => showErrorToast(toast, err, '删除失败'));
  };

  const upload = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    api.drivers.uploadDriver(file)
      .then(() => { toast.show('success', '上传成功'); load(); })
      .catch(showUploadError)
      .finally(() => { e.target.value = ''; });
  };

  const reloadDriver = (item) => {
    if (!item?.id) return;
    setBusyId(item.id);
    api.drivers.reloadDriver(item.id)
      .then(() => {
        toast.show('success', `驱动 ${item.name} 重载成功`);
        load();
      })
      .catch(showReloadError)
      .finally(() => setBusyId(0));
  };

  const fmtSize = (size) => {
    if (!size || size <= 0) return '-';
    return `${(size / 1024).toFixed(1)} KB`;
  };

  const fmtTime = (timeStr) => {
    if (!timeStr) return '-';
    const t = new Date(timeStr);
    if (Number.isNaN(t.getTime())) return '-';
    return t.toLocaleString();
  };

  return (
    <Card
      title="驱动管理"
      extra={
        <div class="flex" style="gap:8px;">
          <button class="btn" onClick={load} disabled={loading()}>
            刷新
          </button>
          <label class="btn btn-primary" style="cursor:pointer;">
            上传驱动
            <input type="file" accept=".wasm" style="display:none" onChange={upload} />
          </label>
        </div>
      }
    >
      <LoadErrorHint error={error()} onRetry={load} />
      {loading() ? (
        <div class="text-center" style="padding:48px; color:var(--text-muted);">
          <div class="loading-spinner" style="margin:0 auto 16px;"></div>
          <div>加载中...</div>
        </div>
      ) : (
        <CrudTable
          style="max-height:520px; overflow:auto;"
          loading={loading()}
          items={items()}
          emptyText="暂无驱动"
          columns={[
            { key: 'id', title: 'ID' },
            { key: 'name', title: '名称' },
            {
              key: 'filename',
              title: '文件',
              render: (d) => d.filename || d.file_path || '',
            },
            {
              key: 'version',
              title: '版本',
              render: (d) => d.version || '-',
            },
            {
              key: 'size',
              title: '大小',
              render: (d) => fmtSize(d.size),
            },
            {
              key: 'loaded',
              title: '运行态',
              render: (d) => (
                <span class={`badge ${d.loaded ? 'badge-running' : 'badge-stopped'}`}>
                  {d.loaded ? '已加载' : '未加载'}
                </span>
              ),
            },
            {
              key: 'last_active',
              title: '最后活跃',
              render: (d) => fmtTime(d.last_active),
            },
          ]}
          renderActions={(d) => (
            <div class="flex" style="gap:8px;">
              <button class="btn" onClick={() => reloadDriver(d)} disabled={busyId() === d.id}>
                {busyId() === d.id ? '重载中...' : '重载'}
              </button>
              <a class="btn" href={`/api/drivers/${d.id}/download`}>下载</a>
              <button class="btn btn-danger" onClick={() => remove(d.id, d.name)}>删除</button>
            </div>
          )}
        />
      )}
    </Card>
  );
}
