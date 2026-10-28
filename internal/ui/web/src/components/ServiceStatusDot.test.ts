import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import ServiceStatusDot from './ServiceStatusDot.svelte';

describe('ServiceStatusDot', () => {
  it('shows the moon for a service idle-suspend put to sleep', () => {
    const { getByTestId, container } = render(ServiceStatusDot, {
      props: { svc: { status: 'inactive', idle_suspended: true } }
    });
    expect(getByTestId('service-moon').getAttribute('title')).toBe('Sleeping');
    expect(container.querySelector('.rounded-full')).toBeNull();
  });

  it('shows the green dot once it is running again, even before the flag clears', () => {
    const { queryByTestId, container } = render(ServiceStatusDot, {
      props: { svc: { status: 'active', idle_suspended: true } }
    });
    expect(queryByTestId('service-moon')).toBeNull();
    expect(container.querySelector('.bg-emerald-500')).toBeTruthy();
  });

  it('keeps the gray dot for a service that is simply stopped', () => {
    const { queryByTestId, container } = render(ServiceStatusDot, {
      props: { svc: { status: 'inactive' } }
    });
    expect(queryByTestId('service-moon')).toBeNull();
    expect(container.querySelector('.bg-gray-300')).toBeTruthy();
  });

  // jsdom has no layout, so the shared footprint is read off the outer box:
  // the same width for both, or a sleeping row's name starts further right.
  it('gives the dot the same footprint as the moon, at either size', () => {
    for (const size of ['sm', 'xs'] as const) {
      const box = (svc: { status: string; idle_suspended?: boolean }) =>
        (render(ServiceStatusDot, { props: { svc, size } }).container.firstElementChild as HTMLElement).className.match(/\bw-\S+ h-\S+/)?.[0];
      expect(box({ status: 'active' })).toBe(box({ status: 'inactive', idle_suspended: true }));
    }
  });
});
