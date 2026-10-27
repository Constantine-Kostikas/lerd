import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';

const { pullPlan, streamPull, streamPullUndo, closeModal, restoreSnapshot } = vi.hoisted(() => ({
  pullPlan: vi.fn(),
  streamPull: vi.fn(),
  streamPullUndo: vi.fn(),
  closeModal: vi.fn(),
  restoreSnapshot: vi.fn()
}));
vi.mock('$stores/worktree', () => ({ pullPlan, streamPull, streamPullUndo }));
vi.mock('$stores/modals', () => ({ closeModal }));
vi.mock('$stores/databases', () => ({ restoreSnapshot }));
const { loadGitCommits, loadUptoFiles, loadUptoVersions } = vi.hoisted(() => ({
  loadGitCommits: vi.fn(),
  loadUptoFiles: vi.fn(),
  loadUptoVersions: vi.fn()
}));
vi.mock('$lib/gitDiff', async (orig) => ({ ...(await orig<object>()), loadGitCommits, loadUptoFiles, loadUptoVersions }));
vi.mock('$components/MonacoDiffEditor.svelte', async () => ({ default: (await import('./MonacoDiffEditor.stub.svelte')).default }));

import PullModal from './PullModal.svelte';

const plan = {
  ahead: 0,
  behind: 3,
  files: 7,
  composer: { label: 'composer install', needed: true, changed: 'composer.lock' },
  js: { label: 'npm ci', needed: false },
  migrate: { label: 'php artisan migrate --force', needed: true },
  migrations_added: 1,
  migrations_missing: 0,
  conflicts: [],
  db: { service: 'mysql', database: 'acme' },
  target: 'abc123',
  branch: 'feature',
  head: 'f00d'
};

const props = { domain: 'acme.test', branch: 'feature', branchLabel: 'feature', path: '/srv/acme' };

function finishWith(done: object) {
  streamPull.mockImplementation(async (_d: string, _b: string, _r: object, _s: object, on: (e: object) => void) => {
    on({ line: 'Fast-forward' });
    on({ done: true, ...done });
  });
}

describe('PullModal', () => {
  beforeEach(() => {
    for (const f of [pullPlan, streamPull, streamPullUndo, closeModal, restoreSnapshot]) f.mockReset();
    pullPlan.mockResolvedValue(plan);
    loadGitCommits.mockReset();
    loadGitCommits.mockResolvedValue({ commits: [], more: false });
  });

  // Every file across the incoming commits, compared up to the commit the plan reviewed.
  it('opens the full list of changed files from the files card', async () => {
    loadUptoFiles.mockResolvedValue([{ path: 'composer.lock', status: 'M' }]);
    loadUptoVersions.mockResolvedValue({ original: 'a', modified: 'b', binary: false, too_large: false, unreadable: false });
    render(PullModal, { props });

    await fireEvent.click(await screen.findByRole('button', { name: /Files changed/ }));
    expect(await screen.findByText('Incoming on feature')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId('diff').dataset.modified).toBe('b'));
    expect(loadUptoFiles).toHaveBeenCalledWith('acme.test', '/srv/acme', 'abc123');
    expect(loadUptoVersions).toHaveBeenCalledWith('acme.test', '/srv/acme', 'abc123', 'composer.lock');

    await fireEvent.keyDown(window, { key: 'Escape' });
    await waitFor(() => expect(screen.queryByText('Incoming on feature')).toBeNull());
    expect(screen.getByRole('button', { name: 'Pull' })).toBeInTheDocument();
  });

  // Read only once the plan's fetch is done, so the list is what the pull brings.
  it('lists the incoming commits after the plan', async () => {
    let resolvePlan: (p: object) => void = () => {};
    pullPlan.mockReturnValue(new Promise((r) => (resolvePlan = r)));
    loadGitCommits.mockResolvedValue({ commits: [{ hash: 'abc1234def', subject: 'feat: incoming' }], more: true });
    render(PullModal, { props });
    await new Promise((r) => setTimeout(r, 0));
    expect(loadGitCommits).not.toHaveBeenCalled();

    resolvePlan(plan);
    await fireEvent.click(await screen.findByRole('button', { name: 'Commits · 1+' }));
    expect(screen.getByText('feat: incoming')).toBeVisible();
    expect(screen.getByText('abc1234')).toBeInTheDocument();
    expect(screen.getByText('Showing the newest 1.')).toBeInTheDocument();
    expect(loadGitCommits).toHaveBeenCalledWith('acme.test', '/srv/acme', 'incoming');
  });

  // The same picture as a branch switch: what comes in, and what it calls for.
  it('shows what the pull brings in and ticks what it calls for', async () => {
    render(PullModal, { props });
    expect(await screen.findByRole('switch', { name: 'composer install' })).toBeChecked();
    expect(screen.getByRole('switch', { name: 'npm ci' })).not.toBeChecked();
    expect(screen.getByRole('switch', { name: 'php artisan migrate --force' })).toBeChecked();
    expect(screen.getByRole('switch', { name: 'Snapshot the database first' })).toBeChecked();
    expect(screen.getByText('composer.lock changed')).toBeInTheDocument();
    expect(screen.getByText('Behind')).toBeInTheDocument();
    expect(pullPlan).toHaveBeenCalledWith('acme.test', 'feature');
  });

  it('pulls with only what stays ticked, then refreshes and closes', async () => {
    finishWith({ ok: true });
    const onDone = vi.fn();
    render(PullModal, { props: { ...props, onDone } });
    await fireEvent.click(await screen.findByRole('switch', { name: 'composer install' }));
    await fireEvent.click(screen.getByRole('button', { name: 'Pull' }));

    expect(streamPull).toHaveBeenCalledWith('acme.test', 'feature', { branch: 'feature', head: 'f00d', target: 'abc123' }, { composer: false, js: false, migrate: true, snapshot: true, isolate: false, stash: false }, expect.any(Function));
    await waitFor(() => expect(closeModal).toHaveBeenCalled());
    expect(onDone).toHaveBeenCalled();
  });

  it('holds the pull back when it would overwrite uncommitted work', async () => {
    pullPlan.mockResolvedValue({ ...plan, conflicts: ['composer.lock'] });
    render(PullModal, { props });
    expect(await screen.findByText(/Your edits to composer\.lock/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Pull' })).toBeDisabled();
  });

  it('pulls over uncommitted work once it may be stashed', async () => {
    finishWith({ ok: true });
    pullPlan.mockResolvedValue({ ...plan, conflicts: ['composer.lock'] });
    render(PullModal, { props });
    await fireEvent.click(await screen.findByRole('switch', { name: 'Stash my changes first' }));
    await fireEvent.click(screen.getByRole('button', { name: 'Pull' }));
    expect(streamPull.mock.calls[0][3]).toMatchObject({ stash: true });
  });

  // The fetch can reveal local commits as well as incoming ones; git would refuse.
  it('holds a diverged pull back, saying why', async () => {
    pullPlan.mockResolvedValue({ ...plan, ahead: 2 });
    render(PullModal, { props });
    expect(await screen.findByText('Diverged: 2 local and 3 remote commits. Merge or rebase it in your editor or terminal.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Pull' })).toBeDisabled();
  });

  // A migration on main's database would change it under the worktrees sharing it.
  it('offers to give sharing worktrees their own database first', async () => {
    finishWith({ ok: true });
    pullPlan.mockResolvedValue({ ...plan, shared_worktrees: ['feature-x', 'bugfix-y'] });
    render(PullModal, { props: { ...props, branch: '' } });
    const row = await screen.findByRole('switch', { name: 'Give worktrees their own database first' });
    expect(row).toBeChecked();
    expect(screen.getByText(/feature-x, bugfix-y share acme/)).toBeInTheDocument();
    await fireEvent.click(screen.getByRole('button', { name: 'Pull' }));
    expect(streamPull.mock.calls[0][3]).toMatchObject({ isolate: true });
  });

  it('leaves the worktrees alone when no migrations come in', async () => {
    pullPlan.mockResolvedValue({ ...plan, migrations_added: 0, migrate: { label: 'php artisan migrate --force', needed: false }, shared_worktrees: ['feature-x'] });
    render(PullModal, { props });
    expect(await screen.findByRole('switch', { name: 'Give worktrees their own database first' })).not.toBeChecked();
  });

  it('says so when there is nothing to pull', async () => {
    pullPlan.mockResolvedValue({ ...plan, behind: 0 });
    render(PullModal, { props });
    expect(await screen.findByText('Up to date')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Pull' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Cancel' })).toBeNull();
    await fireEvent.click(screen.getByRole('button', { name: 'OK' }));
    expect(closeModal).toHaveBeenCalled();
  });

  it('shows why the plan failed', async () => {
    pullPlan.mockResolvedValue({ error: 'this branch has no upstream to pull from' });
    render(PullModal, { props });
    expect(await screen.findByText('this branch has no upstream to pull from')).toBeInTheDocument();
  });

  it('puts the database back when a step after the snapshot failed', async () => {
    finishWith({ ok: false, error: 'migrate failed', snapshot: 'before-switch-feature' });
    restoreSnapshot.mockResolvedValue({ ok: true });
    render(PullModal, { props });
    await screen.findByRole('switch', { name: 'composer install' });
    await fireEvent.click(screen.getByRole('button', { name: 'Pull' }));

    expect(await screen.findByText('migrate failed')).toBeInTheDocument();
    await fireEvent.click(screen.getByRole('button', { name: 'Restore snapshot' }));
    expect(restoreSnapshot).toHaveBeenCalledWith('mysql', 'acme', 'before-switch-feature');
    expect(await screen.findByText('Database restored from before-switch-feature.')).toBeInTheDocument();
  });

  // The pull went through; the clash is explained, not offered a database restore.
  it('stays open to say the stash kept local changes', async () => {
    finishWith({ ok: true, snapshot: 'before-switch-feature', stash: { label: 'lerd: local changes set aside', files: ['composer.lock'] } });
    render(PullModal, { props });
    await screen.findByRole('switch', { name: 'composer install' });
    await fireEvent.click(screen.getByRole('button', { name: 'Pull' }));

    expect(await screen.findByText(/clash with composer\.lock, so they stay in the stash as "lerd: local changes set aside"/)).toBeInTheDocument();
    expect(closeModal).not.toHaveBeenCalled();
    expect(screen.queryByRole('button', { name: 'Restore snapshot' })).toBeNull();
  });

  // Back to where the pull started, the kept work with it; the database stays
  // when no migration ran, so no snapshot is sent.
  it('undoes a pull whose stash was kept', async () => {
    finishWith({ ok: true, moved: true, snapshot: 'branch-main-1', stash: { label: 'lerd: local changes set aside', sha: 'c0ffee', files: ['composer.lock'] } });
    streamPullUndo.mockImplementation(async (_d: string, _b: string, _l: object, _s: object, on: (e: object) => void) => on({ done: true, ok: true }));
    pullPlan.mockResolvedValue({ ...plan, migrate: { label: 'php artisan migrate --force', needed: false }, migrations_added: 0 });
    render(PullModal, { props });
    await screen.findByRole('switch', { name: 'composer install' });
    await fireEvent.click(screen.getByRole('button', { name: 'Pull' }));
    await fireEvent.click(await screen.findByRole('button', { name: 'Undo pull' }));

    expect(streamPullUndo).toHaveBeenCalledWith('acme.test', 'feature', { branch: 'feature', head: 'abc123', target: 'f00d' }, { composer: true, js: false, migrate: false, restore: '', stash: 'c0ffee' }, expect.any(Function));
    await waitFor(() => expect(closeModal).toHaveBeenCalled());
  });

  it('offers no undo when the pull failed before the code moved', async () => {
    finishWith({ ok: false, moved: false, error: 'feature has moved since the review' });
    render(PullModal, { props });
    await screen.findByRole('switch', { name: 'composer install' });
    await fireEvent.click(screen.getByRole('button', { name: 'Pull' }));

    expect(await screen.findByText('feature has moved since the review')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Undo pull' })).toBeNull();
  });

  it('stays open when the undo had to keep work in the stash', async () => {
    finishWith({ ok: true, moved: true, stash: { label: 'lerd: local changes set aside', sha: 'c0ffee', files: ['composer.lock'] } });
    streamPullUndo.mockImplementation(async (_d: string, _b: string, _l: object, _s: object, on: (e: object) => void) =>
      on({ done: true, ok: true, stash: { label: 'lerd: undoing the pull', sha: 'beef', files: ['.lerd.yaml'] } })
    );
    render(PullModal, { props });
    await screen.findByRole('switch', { name: 'composer install' });
    await fireEvent.click(screen.getByRole('button', { name: 'Pull' }));
    await fireEvent.click(await screen.findByRole('button', { name: 'Undo pull' }));

    expect(await screen.findByText(/clash with \.lerd\.yaml, so they stay in the stash as "lerd: undoing the pull"/)).toBeInTheDocument();
    expect(closeModal).not.toHaveBeenCalled();
    expect(screen.queryByRole('button', { name: 'Undo pull' })).toBeNull();
  });
});
