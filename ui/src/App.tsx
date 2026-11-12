import { useEffect, useState } from 'react';
import type { Info, Installer, Options as Choices, Removal } from './bridge';
import { advance, leftForLater, plannedSteps, startRun, type Run } from './run';
import { language } from './i18n';
import { Confirm } from './pages/Confirm';
import { Done } from './pages/Done';
import { Options } from './pages/Options';
import { Progress } from './pages/Progress';
import { Welcome } from './pages/Welcome';
import { WindowCard } from './components/WindowCard';

type Step = 'welcome' | 'options' | 'confirm' | 'progress' | 'done';

// The done page follows the last step after a moment, so the full bar is seen.
const DONE_PAUSE_MS = 600;

export function App({ installer, info, defaults }: { installer: Installer; info: Info; defaults: Choices }) {
  const mode = info.mode;
  const [step, setStep] = useState<Step>(mode === 'install' ? 'welcome' : 'confirm');
  const [choices, setChoices] = useState({ ...defaults, language: language() });
  const [removal, setRemoval] = useState<Removal>({ windowData: false });
  const components = info.components ?? [];
  const fresh = () => startRun(plannedSteps(mode, info.platform, components.length > 0), components.map((c) => c.id));
  const [run, setRun] = useState<Run>(fresh);
  const [skipping, setSkipping] = useState(false);

  useEffect(() => installer.onProgress((event) => setRun((r) => advance(r, event))), [installer]);

  useEffect(() => {
    if (!run.finished) return;
    const id = setTimeout(() => setStep('done'), DONE_PAUSE_MS);
    return () => clearTimeout(id);
  }, [run.finished]);

  function begin() {
    setRun(fresh());
    setSkipping(false);
    setStep('progress');
    void (mode === 'install' ? installer.install(choices) : installer.uninstall(removal));
  }

  const quit = () => void installer.quit();
  const working = step === 'progress' && !run.finished && !run.error;

  return (
    <WindowCard info={info} closable={!working} onMinimise={() => void installer.minimise()} onClose={quit}>
      {page()}
    </WindowCard>
  );

  function page() {
    switch (step) {
      case 'welcome':
        return (
          <Welcome
            info={info}
            onNext={() => setStep('options')}
            onCancel={quit}
            onWebsite={() => void installer.openURL(info.homepage)}
          />
        );
      case 'options':
        return (
          <Options
            installer={installer}
            info={info}
            choices={choices}
            onChange={setChoices}
            onInstall={begin}
            onBack={() => setStep('welcome')}
            onCancel={quit}
          />
        );
      case 'confirm':
        return <Confirm info={info} removal={removal} onChange={setRemoval} onUninstall={begin} onCancel={quit} />;
      case 'progress':
        return (
          <Progress
            info={info}
            run={run}
            onRetry={begin}
            onBack={() => setStep(mode === 'install' ? 'options' : 'confirm')}
            onClose={quit}
            onSkip={() => {
              setSkipping(true);
              void installer.skipComponents();
            }}
            skipping={skipping}
          />
        );
      case 'done':
        return (
          <Done
            info={info}
            later={leftForLater(run).map((id) => ({ id, error: run.fetches[id]!.error }))}
            onLaunch={async () => {
              await installer.launch();
              quit();
            }}
            onClose={quit}
          />
        );
    }
  }
}
