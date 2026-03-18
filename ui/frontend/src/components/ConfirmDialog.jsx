import Modal from './Modal';

function ConfirmDialog(props) {
  return (
    <Modal
      title={props.title || '请确认操作'}
      onClose={props.onCancel}
      closeOnBackdrop={!props.busy}
      contentStyle="width:440px; max-width:92vw;"
    >
      <div class="card-body">
        <div class="text-sm" style="line-height:1.6; color:var(--text-secondary);">
          {props.message}
        </div>
        <div class="modal-actions" style="margin-top:16px;">
          <button
            type="button"
            class="btn btn-outline-primary btn-sm"
            onClick={props.onCancel}
            disabled={props.busy}
          >
            取消
          </button>
          <button
            type="button"
            class={`btn btn-sm ${props.variant === 'danger' ? 'btn-outline-danger' : 'btn-primary'}`}
            onClick={props.onConfirm}
            disabled={props.busy}
          >
            {props.confirmText || '确认'}
          </button>
        </div>
      </div>
    </Modal>
  );
}

export default ConfirmDialog;
