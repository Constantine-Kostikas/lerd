// Scrolls a truncated text to its end and back while it is hovered or focused, so a
// long name in a narrow tab reads without widening it. The parent triggers it, since
// the text is often only part of the button, or a further ancestor named by selector.
export function marquee(node: HTMLElement, trigger?: string) {
  const host = (trigger ? node.closest<HTMLElement>(trigger) : node.parentElement) ?? node;
  const start = () => {
    const overflow = node.scrollWidth - node.clientWidth;
    if (overflow <= 0) return;
    node.style.setProperty('--marquee-shift', `-${overflow}px`);
    // ~40px a second reads comfortably; short overflows still get a beat.
    node.style.setProperty('--marquee-duration', `${Math.max(1.5, overflow / 40)}s`);
    node.classList.add('lerd-marquee');
  };
  const stop = () => node.classList.remove('lerd-marquee');
  const events: [string, () => void][] = [
    ['mouseenter', start],
    ['mouseleave', stop],
    ['focusin', start],
    ['focusout', stop]
  ];
  for (const [type, fn] of events) host.addEventListener(type, fn);
  return {
    destroy() {
      for (const [type, fn] of events) host.removeEventListener(type, fn);
    }
  };
}
