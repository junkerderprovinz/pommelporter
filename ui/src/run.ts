// The progress page's model: the steps a run is expected to take and how far
// the "progress" events have moved it. A platform skips the steps that mean
// nothing to it and may report one this list does not know, so the list is
// a forecast that the events correct.
import type { FetchState, Progress } from './bridge';

export type Mode = 'install' | 'uninstall';

export type StepState = 'waiting' | 'running' | 'done' | 'failed';

/** One component's fetch as the page shows it. */
export interface Fetch {
  state: 'waiting' | FetchState;
  done: number;
  /** -1 while unknown. */
  total: number;
  error: string;
  reason: string;
}

export interface Run {
  steps: string[];
  /** The running step's index, -1 before the first event. */
  current: number;
  finished: boolean;
  error: string | null;
  /** The components to fetch by id, in the order they are fetched. */
  fetches: Record<string, Fetch>;
}

/** The step that fetches the components, last of an install. */
export const COMPONENTS_STEP = 'components';

/** The step names of internal/setup, in the order the platforms run them. */
export function plannedSteps(mode: Mode, platform: string, components = false): string[] {
  const linux = platform === 'linux';
  if (mode === 'uninstall') {
    return linux ? ['remove'] : ['close', 'remove'];
  }
  const steps = linux
    ? ['prepare', 'files', 'shortcuts', 'register']
    : ['prepare', 'close', 'legacy', 'files', 'shortcuts', 'register', 'updates'];
  return components ? [...steps, COMPONENTS_STEP] : steps;
}

export function startRun(steps: string[], components: string[] = []): Run {
  const fetches: Record<string, Fetch> = {};
  for (const id of components) fetches[id] = { state: 'waiting', done: 0, total: -1, error: '', reason: '' };
  return { steps, current: -1, finished: false, error: null, fetches };
}

export function advance(run: Run, event: Progress): Run {
  if ('error' in event && !('component' in event)) {
    // A refusal before the first step still fails a row, so the list shows where it stopped.
    return { ...run, current: Math.max(run.current, 0), error: event.error };
  }
  if ('component' in event) {
    const was = run.fetches[event.component] ?? { state: 'waiting', done: 0, total: -1, error: '', reason: '' };
    const fetch: Fetch = {
      state: event.state,
      done: event.done ?? was.done,
      total: event.total ?? was.total,
      error: event.error ?? '',
      reason: event.reason ?? '',
    };
    return { ...run, fetches: { ...run.fetches, [event.component]: fetch } };
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

/** How much of the components' work is over, from 0 to 1. A download counts by its bytes while it runs. */
function fetched(run: Run): number {
  const all = Object.values(run.fetches);
  if (all.length === 0) return 0;
  const share = (f: Fetch) => {
    if (f.state === 'waiting') return 0;
    if (f.state !== 'running') return 1;
    return f.total > 0 ? Math.min(f.done / f.total, 1) : 0;
  };
  return all.reduce((sum, f) => sum + share(f), 0) / all.length;
}

/** How much of the bar is filled, from 0 to 1. A step counts once it is over. */
export function fraction(run: Run): number {
  if (run.finished) return 1;
  const within = run.steps[run.current] === COMPONENTS_STEP ? fetched(run) : 0;
  return (Math.max(run.current, 0) + within) / run.steps.length;
}

/** Whether the components step runs, which is when it can be skipped. */
export function fetching(run: Run): boolean {
  return !run.finished && run.error === null && run.steps[run.current] === COMPONENTS_STEP;
}

/** A component the app fetches itself, since the installer could not or was told to skip it. */
export type Later = Fetch & { id: string };

export function leftForLater(run: Run): Later[] {
  return Object.entries(run.fetches)
    .filter(([, f]) => f.state === 'failed' || f.state === 'skipped')
    .map(([id, f]) => ({ ...f, id }));
}
