import { getErrorMessage } from '../api/errorMessages';

export function showErrorToast(toast, error, fallback = '操作失败') {
  if (!toast || typeof toast.show !== 'function') return;
  toast.show('error', getErrorMessage(error, fallback));
}

