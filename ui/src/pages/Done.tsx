import { useState } from 'react';
import type { Info } from '../bridge';
import { t } from '../i18n';
import { componentName, nameList } from '../fetches';
import { stays } from '../stays';
import { Button } from '../glimstone/react/Button';
import { Page } from '../components/Page';

export function Done({
  info,
  later,
  onLaunch,
  onClose,
}: {
  info: Info;
  /** The ids of the components the app fetches itself, with why the installer did not. */
  later: { id: string; error: string }[];
  onLaunch: () => Promise<void>;
  onClose: () => void;
}) {
  const [launchError, setLaunchError] = useState('');
  const installed = info.mode === 'install';
  const names = later.map(({ id }) => {
    const c = info.components?.find((c) => c.id === id);
    return c ? componentName(c) : id;
  });
  const reasons = later.flatMap(({ error }, i) => (error ? [`${names[i]}: ${error}`] : []));

  return (
    <Page
      actions={
        installed ? (
          <>
            <Button label={t('button.close')} labelKey="button.close" tone="neutral" className="glim-btn-key" onClick={onClose} />
            <Button
              label={t('button.start', { name: info.name })}
              labelKey="button.start"
              tone="accent"
              className="glim-btn-key"
              autoFocus
              onClick={() => onLaunch().catch((e: unknown) => setLaunchError(String(e instanceof Error ? e.message : e)))}
            />
          </>
        ) : (
          <Button label={t('button.close')} labelKey="button.close" tone="accent" className="glim-btn-key" autoFocus onClick={onClose} />
        )
      }
    >
      <div className="flex flex-1 flex-col items-center justify-center gap-4 text-center">
        <p className="text-title font-semibold text-carbon-text">
          {t(installed ? 'done.installed' : 'done.uninstalled', { name: info.name })}
        </p>
        {!installed && <p className="max-w-md text-body text-carbon-textSub">{stays(info, 'note')}</p>}
        {installed && names.length > 0 && (
          <div className="glim-well flex max-w-md flex-col gap-1 px-4 py-2.5 text-start">
            <p className="text-sm text-carbon-text">{t('done.later', { name: info.name, list: nameList(names) })}</p>
            {/* The app reports in English, so each line keeps its own direction. */}
            {reasons.map((r) => (
              <p key={r} dir="auto" className="max-h-10 overflow-y-auto text-subline break-words text-carbon-textSub select-text">
                {r}
              </p>
            ))}
          </div>
        )}
        {launchError && (
          <p role="alert" dir="auto" className="max-w-md text-subline text-statusFail">
            {launchError}
          </p>
        )}
      </div>
    </Page>
  );
}
