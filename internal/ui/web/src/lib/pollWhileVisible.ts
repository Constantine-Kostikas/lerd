// UNFOCUSED_MS is how often a poll still runs while its window is open but
// another one has focus, such as a dashboard left on a second monitor.
export const UNFOCUSED_MS = 60_000;

// pollWhileVisible runs fn now and every intervalMs, skipping ticks while the
// tab is hidden, slowing to UNFOCUSED_MS while the window is unfocused, and
// catching up the moment it shows or takes focus again. Returns the stop.
export function pollWhileVisible(fn: () => void, intervalMs: number): () => void {
  let stopped = false;
  let last = 0;
  function tick() {
    if (stopped) return;
    if (typeof document !== 'undefined') {
      if (document.hidden) return;
      if (!document.hasFocus() && Date.now() - last < UNFOCUSED_MS) return;
    }
    last = Date.now();
    fn();
  }
  tick();
  const timer = setInterval(tick, intervalMs);

  const onVisibility = () => {
    if (typeof document === 'undefined') return;
    if (!document.hidden) tick();
  };
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisibility);
  }
  if (typeof window !== 'undefined') {
    window.addEventListener('focus', tick);
  }

  return () => {
    stopped = true;
    clearInterval(timer);
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', onVisibility);
    }
    if (typeof window !== 'undefined') {
      window.removeEventListener('focus', tick);
    }
  };
}
