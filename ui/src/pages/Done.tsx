import { useState } from 'react';
import type { Info } from '../bridge';
import { t } from '../i18n';
import { Button } from '../glimstone/react/Button';
import { Page } from '../components/Page';

export function Done({ info, onLaunch, onClose }: { info: Info; onLaunch: () => Promise<void>; onClose: () => void }) {
  const [launchError, setLaunchError] = useState('');
  const installed = info.mode === 'install';

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
        {!installed && <p className="max-w-md text-body text-carbon-textSub">{t('done.kept')}</p>}
        {launchError && (
          <p role="alert" dir="auto" className="max-w-md text-subline text-statusFail">
            {launchError}
          </p>
        )}
      </div>
    </Page>
  );
}
