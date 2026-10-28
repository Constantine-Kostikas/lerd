import { fetchEvent, type Lens, type LensGroup } from '$lib/lens';
import type { DumpEvent } from '$lib/dumpEvent';
import { tick } from 'svelte';

// Which lens rows are open, and the whole event each open row read: the list
// leaves out the call stack and a mail's HTML.
export class LensRows {
  expanded = $state<Record<string, boolean>>({});
  details = $state<Record<string, DumpEvent>>({});

  toggle(id: string) {
    this.expanded[id] = !this.expanded[id];
    if (!this.expanded[id]) delete this.details[id];
    else if (!this.details[id]) fetchEvent(id).then((e) => (this.expanded[id] ? (this.details[id] = e) : undefined), () => {});
  }

  // reveal opens the row the timeline sent the user to and scrolls to it, once
  // the page holds it, saying whether it did.
  reveal(groups: LensGroup[], id: string): boolean {
    if (!groups.some((g) => g.rows.some((r) => r.event.id === id))) return false;
    if (!this.expanded[id]) this.toggle(id);
    scrollToEvent(id);
    return true;
  }

  // A row that leaves the page lets go of its whole event, mail HTML included,
  // and closes, so coming back it opens and reads it again.
  keepOnly(groups: LensGroup[]) {
    const held = new Set(groups.flatMap((g) => g.rows.map((r) => r.event.id)));
    for (const id of Object.keys(this.details)) {
      if (held.has(id)) continue;
      delete this.details[id];
      delete this.expanded[id];
    }
  }
}

// revealOrLoad opens the row a timeline sent the user to, or reads more of the
// lens when its page does not hold it yet: the rest of a cut group first, then
// the next page. 'missing' means there is nothing left to read.
export function revealOrLoad(
  rows: LensRows,
  lens: Pick<Lens, 'loadRows' | 'loadMore'>,
  groups: LensGroup[],
  more: boolean,
  id: string
): 'shown' | 'loading' | 'missing' {
  if (rows.reveal(groups, id)) return 'shown';
  if (groups.length === 0) return 'loading';
  const partial = groups.find((g) => g.rows.length < g.count);
  if (partial) {
    void lens.loadRows(partial.key);
    return 'loading';
  }
  if (!more) return 'missing';
  void lens.loadMore();
  return 'loading';
}

// scrollToEvent brings a lens row into view once the DOM has drawn it.
export function scrollToEvent(id: string) {
  void tick().then(() =>
    [...document.querySelectorAll<HTMLElement>('[data-event-id]')].find((el) => el.dataset.eventId === id)?.scrollIntoView?.({ block: 'center' })
  );
}

export function localTime(ts: string): string {
  const d = new Date(ts);
  return isNaN(d.getTime()) ? ts : d.toLocaleTimeString();
}
