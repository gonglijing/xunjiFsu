import { createSignal } from 'solid-js';

export function usePageLoader(loader, options = {}) {
  const [loading, setLoading] = createSignal(options.initialLoading ?? true);
  const [error, setError] = createSignal('');

  const run = async (...args) => {
    setLoading(true);
    setError('');

    try {
      return await loader(...args);
    } catch (err) {
      if (typeof options.onError === 'function') {
        options.onError(err);
      }

      if (typeof options.errorMessage === 'function') {
        const message = options.errorMessage(err);
        if (message) setError(`${message}`);
      } else if (typeof options.errorMessage === 'string' && options.errorMessage) {
        setError(options.errorMessage);
      }

      return null;
    } finally {
      setLoading(false);
    }
  };

  return {
    loading,
    error,
    setError,
    run,
  };
}
