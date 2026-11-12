import type { Info } from '../bridge';
import { t, type Key } from '../i18n';
import { componentName, sizeText } from '../fetches';
import { COMPONENTS_STEP, fetching, fraction, stepState, type Fetch, type Mode, type Run, type StepState } from '../run';
import { Button } from '../glimstone/react/Button';
import { IconCheck, IconClose, IconLater } from '../components/icons';
import { Page } from '../components/Page';

/** The words for a step; a few name the opposite act when uninstalling. */
function stepLabel(step: string, mode: Mode, name: string): string {
  const keys: Record<string, Key> = {
    prepare: 'step.prepare',
    close: 'step.close',
    legacy: 'step.legacy',
    files: 'step.files',
    shortcuts: mode === 'install' ? 'step.shortcuts.install' : 'step.shortcuts.uninstall',
    register: mode === 'install' ? 'step.register.install' : 'step.register.uninstall',
    updates: mode === 'install' ? 'step.updates.install' : 'step.updates.uninstall',
    remove: 'step.remove',
    [COMPONENTS_STEP]: 'step.components',
  };
  return t(keys[step] ?? 'step.other', { name });
}

type Mark = StepState | 'later';

function StepMark({ state }: { state: Mark }) {
  switch (state) {
    case 'done':
      return <IconCheck className="text-statusOk" width={14} height={14} />;
    case 'failed':
      return <IconClose className="text-statusFail" width={12} height={12} />;
    case 'later':
      return <IconLater className="text-statusWarn" width={14} height={14} />;
    case 'running':
      return <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-accent border-t-transparent" />;
    default:
      return <span className="h-1.5 w-1.5 rounded-full bg-carbon-surface3" />;
  }
}

const LABEL_INK: Record<Mark, string> = {
  waiting: 'text-carbon-textMuted',
  running: 'text-carbon-text font-medium',
  done: 'text-carbon-textSub',
  failed: 'text-statusFail font-medium',
  later: 'text-carbon-textSub',
};

/** A component's row mark: one the app fetches later is not a failed install. */
function fetchMark(f: Fetch): Mark {
  switch (f.state) {
    case 'running':
      return 'running';
    case 'done':
    case 'present':
      return 'done';
    case 'failed':
    case 'skipped':
      return 'later';
    default:
      return 'waiting';
  }
}

function fetchDetail(f: Fetch): string {
  switch (f.state) {
    case 'running':
      return sizeText(f);
    case 'present':
      return t('component.present');
    case 'failed':
    case 'skipped':
      return t('component.later');
    default:
      return '';
  }
}

export function Progress({
  info,
  run,
  onRetry,
  onBack,
  onClose,
  onSkip,
  skipping,
}: {
  info: Info;
  run: Run;
  onRetry: () => void;
  onBack: () => void;
  onClose: () => void;
  onSkip: () => void;
  /** Skip was pressed and the download under way is being stopped. */
  skipping: boolean;
}) {
  const mode = info.mode;
  const percent = Math.round(fraction(run) * 100);
  const failed = run.error !== null;
  const components = info.components ?? [];

  return (
    <Page
      actions={
        failed ? (
          <>
            <Button label={t('button.close')} labelKey="button.close" tone="neutral" className="glim-btn-key" onClick={onClose} />
            <Button label={t('button.back')} labelKey="button.back" tone="neutral" className="glim-btn-key" onClick={onBack} />
            <Button label={t('button.retry')} labelKey="button.retry" tone="accent" className="glim-btn-key" onClick={onRetry} />
          </>
        ) : (
          fetching(run) && (
            <Button
              label={t('button.skip')}
              labelKey="button.skip"
              tone="neutral"
              className="glim-btn-key"
              disabled={skipping}
              busy={skipping}
              onClick={onSkip}
            />
          )
        )
      }
    >
      <div className="my-auto flex flex-col gap-4">
        <div
          role="progressbar"
          aria-label={t('progress.label')}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={percent}
          className="glim-meter h-2 shrink-0 overflow-hidden rounded-pill bg-carbon-surface2"
        >
          <div
            className={`h-full rounded-pill ease-out motion-safe:transition-[width] motion-safe:duration-500 ${failed ? 'bg-statusFailSolid' : 'bg-accent'}`}
            style={{ width: `${percent}%` }}
          />
        </div>
        <ol className="flex flex-col">
          {run.steps.map((step, i) => {
            const state = stepState(run, i);
            return (
              <li key={step} className="flex flex-col text-sm" aria-current={state === 'running' ? 'step' : undefined}>
                <span className="flex items-center gap-3">
                  <span className="flex h-4 w-4 shrink-0 items-center justify-center">
                    <StepMark state={state} />
                  </span>
                  <span className={LABEL_INK[state]}>{stepLabel(step, mode, info.name)}</span>
                </span>
                {step === COMPONENTS_STEP && components.length > 0 && (
                  <ul className="flex flex-col ps-7">
                    {components.map((c) => {
                      const f = run.fetches[c.id] ?? { state: 'waiting', done: 0, total: -1, error: '', reason: '' };
                      const mark = fetchMark(f);
                      return (
                        <li key={c.id} className="flex items-center gap-3 text-subline">
                          <span className="flex h-4 w-4 shrink-0 items-center justify-center">
                            <StepMark state={mark} />
                          </span>
                          <span className={LABEL_INK[mark]}>{componentName(c)}</span>
                          <span className="flex-1" />
                          <span className="text-carbon-textMuted tabular-nums">{fetchDetail(f)}</span>
                        </li>
                      );
                    })}
                  </ul>
                )}
              </li>
            );
          })}
        </ol>
        {failed && (
          <div role="alert" className="glim-well flex shrink-0 flex-col gap-1 px-4 py-2.5">
            <p className="text-sm font-medium text-carbon-text">
              {t(mode === 'install' ? 'progress.failed.install' : 'progress.failed.uninstall', { name: info.name })}
            </p>
            {/* The platform reports in English, so the sentence keeps its own direction. */}
            <p dir="auto" className="max-h-10 overflow-y-auto text-subline break-words text-carbon-textSub select-text">
              {run.error}
            </p>
          </div>
        )}
      </div>
    </Page>
  );
}
