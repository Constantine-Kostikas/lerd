import { render, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import RequestTimeline from './RequestTimeline.svelte';

const waterfall = {
  total: 100,
  rows: [
    { label: 'GET /cart', layer: 'request', start: 0, end: 100, note: '' },
    { label: 'GET https://api.test', layer: 'error', start: 10, end: 40, note: '', id: 'ev1', lens: 'http' }
  ]
} as never;

describe('RequestTimeline', () => {
  it('opens a row in its own lens from the popover', async () => {
    const onopen = vi.fn();
    const { getByText, getByLabelText } = render(RequestTimeline, { props: { waterfall, onopen } });
    await fireEvent.click(getByText('GET https://api.test'));
    await fireEvent.click(getByLabelText('Details'));
    expect(onopen).toHaveBeenCalledWith('http', 'ev1');
  });

  it('offers nothing to open for the request itself', async () => {
    const { getByText, queryByLabelText } = render(RequestTimeline, { props: { waterfall, onopen: vi.fn() } });
    await fireEvent.click(getByText('GET /cart'));
    expect(queryByLabelText('Details')).toBeNull();
  });
});
