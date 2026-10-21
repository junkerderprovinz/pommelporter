import { useEffect, useState } from 'react';
import type { Info, Installer, Options as Choices } from './bridge';
import { advance, plannedSteps, startRun, type Run } from './run';
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
  const [run, setRun] = useState<Run>(() => startRun(plannedSteps(mode, info.platform)));

  useEffect(() => installer.onProgress((event) => setRun((r) => advance(r, event))), [installer]);

  useEffect(() => {
    if (!run.finished) return;
    const id = setTimeout(() => setStep('done'), DONE_PAUSE_MS);
    return () => clearTimeout(id);
  }, [run.finished]);

  function begin() {
    setRun(startRun(plannedSteps(mode, info.platform)));
    setStep('progress');
    void (mode === 'install' ? installer.install(choices) : installer.uninstall());
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
        return <Confirm info={info} onUninstall={begin} onCancel={quit} />;
      case 'progress':
        return (
          <Progress
            info={info}
            run={run}
            onRetry={begin}
            onBack={() => setStep(mode === 'install' ? 'options' : 'confirm')}
            onClose={quit}
          />
        );
      case 'done':
        return (
          <Done
            info={info}
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
