// UNFOCUSED_MS is how often a poll still runs while its window is open but
// another one has focus, such as a dashboard left on a second monitor.
export const UNFOCUSED_MS = 60_000;

// pollWhileVisible runs fn now and every intervalMs, skipping ticks while the
// tab is hidden, slowing to UNFOCUSED_MS while the window is unfocused, and
// catching up the moment it shows or takes focus again. Returns the stop.
export function pollWhileVisible(fn: () => void, intervalMs: number): () => void {
  let stopped = false;
  let last = 0;
  function run(focused: boolean) {
    if (stopped) return;
    if (typeof document !== 'undefined' && document.hidden) return;
    if (!focused && Date.now() - last < UNFOCUSED_MS) return;
    last = Date.now();
    fn();
  }
  const tick = () => run(typeof document === 'undefined' || document.hasFocus());
  // The event is the word on focus: inside it hasFocus() can still say no.
  const onFocus = () => run(true);
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
    window.addEventListener('focus', onFocus);
  }

  return () => {
    stopped = true;
    clearInterval(timer);
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', onVisibility);
    }
    if (typeof window !== 'undefined') {
      window.removeEventListener('focus', onFocus);
    }
  };
}
