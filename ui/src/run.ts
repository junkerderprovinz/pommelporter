// The progress page's model: the steps a run is expected to take and how far
// the "progress" events have moved it. A platform skips the steps that mean
// nothing to it and may report one this list does not know, so the list is
// a forecast that the events correct.
import type { Progress } from './bridge';

export type Mode = 'install' | 'uninstall';

export type StepState = 'waiting' | 'running' | 'done' | 'failed';

export interface Run {
  steps: string[];
  /** The running step's index, -1 before the first event. */
  current: number;
  finished: boolean;
  error: string | null;
}

/** The step names of internal/setup, in the order the platforms run them. */
export function plannedSteps(mode: Mode, platform: string): string[] {
  const linux = platform === 'linux';
  if (mode === 'uninstall') {
    return linux
      ? ['prepare', 'shortcuts', 'remove']
      : ['prepare', 'close', 'updates', 'shortcuts', 'register', 'remove'];
  }
  return linux
    ? ['prepare', 'files', 'shortcuts']
    : ['prepare', 'close', 'legacy', 'files', 'shortcuts', 'register', 'updates'];
}

export function startRun(steps: string[]): Run {
  return { steps, current: -1, finished: false, error: null };
}

export function advance(run: Run, event: Progress): Run {
  if ('error' in event) {
    // A refusal before the first step still fails a row, so the list shows where it stopped.
    return { ...run, current: Math.max(run.current, 0), error: event.error };
  }
  if (event.step === 'done') {
    return { ...run, current: run.steps.length, finished: true };
  }
  const at = run.steps.indexOf(event.step);
  if (at >= 0) return { ...run, current: at };
  const next = run.current + 1;
  return {
    ...run,
    steps: [...run.steps.slice(0, next), event.step, ...run.steps.slice(next)],
    current: next,
  };
}

/** A step the run passed without reporting it counts as done: there was nothing to do. */
export function stepState(run: Run, index: number): StepState {
  if (run.finished || index < run.current) return 'done';
  if (index > run.current) return 'waiting';
  return run.error === null ? 'running' : 'failed';
}

/** How much of the bar is filled, from 0 to 1. A step counts once it is over. */
export function fraction(run: Run): number {
  if (run.finished) return 1;
  return Math.max(run.current, 0) / run.steps.length;
}
