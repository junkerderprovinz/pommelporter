import { describe, expect, it } from 'vitest';
import { advance, fraction, plannedSteps, startRun, stepState } from './run';

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

describe('the planned steps', () => {
  it('leave out what Linux does not do', () => {
    expect(plannedSteps('install', 'linux')).not.toContain('updates');
    expect(plannedSteps('install', 'windows')).toContain('updates');
  });

  it('end uninstalling with the files', () => {
    expect(plannedSteps('uninstall', 'windows').at(-1)).toBe('remove');
    expect(plannedSteps('uninstall', 'linux').at(-1)).toBe('remove');
  });
});
