function Modal(props) {
  const handleBackdropClick = (event) => {
    if (!props.closeOnBackdrop) return;
    if (event.target !== event.currentTarget) return;
    props.onClose?.();
  };

  return (
    <div class="modal-backdrop" style={props.backdropStyle} onClick={handleBackdropClick}>
      <div class="card" style={props.contentStyle}>
        {(props.title || props.onClose) && (
          <div class="card-header">
            {props.title && <h3 class="card-title">{props.title}</h3>}
            {props.onClose && (
              <button class="btn btn-ghost btn-no-icon btn-only-icon btn-close-lite" onClick={props.onClose}>
                ✕
              </button>
            )}
          </div>
        )}
        {props.children}
      </div>
    </div>
  );
}

export default Modal;
