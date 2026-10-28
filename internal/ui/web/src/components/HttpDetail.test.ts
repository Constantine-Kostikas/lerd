import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import HttpDetail from './HttpDetail.svelte';

describe('HttpDetail', () => {
  it('splits headers into name and value and pretty-prints a JSON body', () => {
    const { container } = render(HttpDetail, {
      props: { label: 'Response', detail: { headers: ['Content-Type: application/json', 'X-Trace: abc'], body: '{"error":"No such customer"}' } }
    });
    const cells = [...container.querySelectorAll('td')].map((td) => td.textContent);
    expect(cells).toEqual(['Content-Type', 'application/json', 'X-Trace', 'abc']);
    expect(container.querySelector('pre')?.textContent).toBe('{\n  "error": "No such customer"\n}');
  });

  it('shows a body that is not JSON as it came and says when it was cut', () => {
    const { container } = render(HttpDetail, { props: { label: 'Response', detail: { headers: [], body: 'a=1&b=2', truncated: true } } });
    expect(container.querySelector('pre')?.textContent).toBe('a=1&b=2');
    expect(container.textContent).toContain('16 KB');
  });

  it('renders nothing for a half with no headers and no body', () => {
    const { container } = render(HttpDetail, { props: { label: 'Request', detail: { headers: [] } } });
    expect(container.textContent?.trim()).toBe('');
  });
});
