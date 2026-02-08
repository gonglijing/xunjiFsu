import { createSignal, onMount, onCleanup, createEffect } from 'solid-js';
import api from '../api/services';
import { onUnauthorized } from '../api';
import { getAuthCheckIntervalMs } from './runtimeConfig';

const AUTH_CHECK_INTERVAL_MS = getAuthCheckIntervalMs();

export function useAuthGuard(pathAccessor, navigate) {
  const [authed, setAuthed] = createSignal(true);
  let inFlightCheck = null;
  let lastCheckAt = 0;

  const gotoLogin = () => {
    setAuthed(false);
    inFlightCheck = null;
    lastCheckAt = 0;
    navigate('/login', { replace: true });
  };

  onMount(() => {
    const unsubscribe = onUnauthorized(() => {
      gotoLogin();
      return true;
    });
    onCleanup(unsubscribe);
  });

  const checkAuth = async () => {
    const currentPath = pathAccessor();
    if (currentPath === '/login') {
      setAuthed(true);
      return true;
    }

    if (inFlightCheck) {
      return inFlightCheck;
    }

    if (Date.now() - lastCheckAt < AUTH_CHECK_INTERVAL_MS) {
      return authed();
    }

    inFlightCheck = (async () => {
      try {
        await api.status.getStatus();
        setAuthed(true);
        return true;
      } catch {
        gotoLogin();
        return false;
      } finally {
        lastCheckAt = Date.now();
        inFlightCheck = null;
      }
    })();

    return inFlightCheck;
  };

  createEffect(() => {
    pathAccessor();
    checkAuth();
  });

  return {
    authed,
    checkAuth,
  };
}
