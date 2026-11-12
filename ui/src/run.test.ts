import { describe, expect, it } from 'vitest';
import { advance, fetching, fraction, leftForLater, plannedSteps, startRun, stepState } from './run';

const steps = () => startRun(['prepare', 'files', 'shortcuts']);

describe('a run', () => {
  it('waits on every step before the first event', () => {
    const run = steps();
    expect([0, 1, 2].map((i) => stepState(run, i))).toEqual(['waiting', 'waiting', 'waiting']);
    expect(fraction(run)).toBe(0);
  });

  it('marks the reported step running and the ones before it done', () => {
    const run = advance(steps(), { step: 'files' });
    expect([0, 1, 2].map((i) => stepState(run, i))).toEqual(['done', 'running', 'waiting']);
    expect(fraction(run)).toBeCloseTo(1 / 3);
  });

  it('fills the bar and every row when done arrives', () => {
    const run = advance(advance(steps(), { step: 'prepare' }), { step: 'done' });
    expect(run.finished).toBe(true);
    expect([0, 1, 2].map((i) => stepState(run, i))).toEqual(['done', 'done', 'done']);
    expect(fraction(run)).toBe(1);
  });

  it('fails the running step on an error', () => {
    const run = advance(advance(steps(), { step: 'files' }), { error: 'disk full' });
    expect(stepState(run, 1)).toBe('failed');
    expect(stepState(run, 2)).toBe('waiting');
    expect(run.error).toBe('disk full');
  });

  it('fails the first step on an error before any step started', () => {
    const run = advance(steps(), { error: 'refused' });
    expect(stepState(run, 0)).toBe('failed');
  });

  it('inserts a step it did not plan after the current one', () => {
    const run = advance(advance(steps(), { step: 'prepare' }), { step: 'icons' });
    expect(run.steps).toEqual(['prepare', 'icons', 'files', 'shortcuts']);
    expect(stepState(run, 1)).toBe('running');
  });
});

describe('the components', () => {
  const fetch = () => advance(startRun(['files', 'components'], ['a', 'b']), { step: 'components' });

  it('fill the last step of the bar by the bytes that arrived', () => {
    const run = advance(fetch(), { component: 'a', state: 'running', done: 50, total: 100 });
    expect(fraction(run)).toBeCloseTo((1 + 0.25) / 2);
    expect(fetching(run)).toBe(true);
  });

  it('do not fail the install when one of them fails', () => {
    const run = advance(fetch(), { component: 'a', state: 'failed', error: 'no such host' });
    expect(run.error).toBeNull();
    expect(stepState(run, 1)).toBe('running');
    expect(run.fetches.a!.error).toBe('no such host');
  });

  it('leave the failed and the skipped ones for the app', () => {
    let run = advance(fetch(), { component: 'a', state: 'failed', error: 'timeout' });
    run = advance(run, { component: 'b', state: 'skipped' });
    run = advance(run, { step: 'done' });
    expect(leftForLater(run)).toEqual(['a', 'b']);
    expect(fetching(run)).toBe(false);
  });

  it('count a copy that was there already as done', () => {
    const run = advance(advance(fetch(), { component: 'a', state: 'present' }), { component: 'b', state: 'done' });
    expect(leftForLater(run)).toEqual([]);
    expect(fraction(run)).toBe(1);
  });
});

describe('the planned steps', () => {
  it('end an install with the components when there are some', () => {
    expect(plannedSteps('install', 'windows', true).at(-1)).toBe('components');
    expect(plannedSteps('install', 'linux', false)).not.toContain('components');
    expect(plannedSteps('uninstall', 'windows', true)).not.toContain('components');
  });

  it('leave out what Linux does not do', () => {
    expect(plannedSteps('install', 'linux')).not.toContain('updates');
    expect(plannedSteps('install', 'windows')).toContain('updates');
  });

  it('end uninstalling with the files', () => {
    expect(plannedSteps('uninstall', 'windows').at(-1)).toBe('remove');
    expect(plannedSteps('uninstall', 'linux').at(-1)).toBe('remove');
  });
});
