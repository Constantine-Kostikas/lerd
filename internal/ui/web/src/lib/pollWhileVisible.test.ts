import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest';
import { pollWhileVisible, UNFOCUSED_MS } from './pollWhileVisible';

function setHidden(hidden: boolean) {
  Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden });
}

function setFocused(focused: boolean) {
  vi.spyOn(document, 'hasFocus').mockReturnValue(focused);
}

describe('pollWhileVisible', () => {
  beforeEach(() => setFocused(true));

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    setHidden(false);
  });

  it('slows to one tick a minute while another window has focus', () => {
    vi.useFakeTimers();
    setFocused(false);
    const fn = vi.fn();
    const stop = pollWhileVisible(fn, 1000);
    vi.advanceTimersByTime(UNFOCUSED_MS - 1000);
    expect(fn).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(1000);
    expect(fn).toHaveBeenCalledTimes(2);
    stop();
  });

  it('catches up the moment its window takes focus again', () => {
    vi.useFakeTimers();
    setFocused(false);
    const fn = vi.fn();
    const stop = pollWhileVisible(fn, 1000);
    vi.advanceTimersByTime(5000);
    setFocused(true);
    window.dispatchEvent(new Event('focus'));
    expect(fn).toHaveBeenCalledTimes(2);
    vi.advanceTimersByTime(1000);
    expect(fn).toHaveBeenCalledTimes(3);
    stop();
  });

  it('runs at once and on every interval', () => {
    vi.useFakeTimers();
    const fn = vi.fn();
    const stop = pollWhileVisible(fn, 1000);
    vi.advanceTimersByTime(2000);
    expect(fn).toHaveBeenCalledTimes(3);
    stop();
  });

  it('skips ticks while hidden and catches up when the tab shows', () => {
    vi.useFakeTimers();
    const fn = vi.fn();
    const stop = pollWhileVisible(fn, 1000);
    setHidden(true);
    vi.advanceTimersByTime(3000);
    expect(fn).toHaveBeenCalledTimes(1);
    setHidden(false);
    document.dispatchEvent(new Event('visibilitychange'));
    expect(fn).toHaveBeenCalledTimes(2);
    stop();
  });

  it('stops ticking and listening once stopped', () => {
    vi.useFakeTimers();
    const fn = vi.fn();
    pollWhileVisible(fn, 1000)();
    vi.advanceTimersByTime(3000);
    document.dispatchEvent(new Event('visibilitychange'));
    expect(fn).toHaveBeenCalledTimes(1);
  });
});
