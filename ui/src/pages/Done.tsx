import { useState } from 'react';
import type { Info } from '../bridge';
import { t } from '../i18n';
import { componentName, nameList, reasonText } from '../fetches';
import type { Later } from '../run';
import { stays } from '../stays';
import { Button } from '../glimstone/react/Button';
import { IconExpand } from '../glimstone/react/glyphs';
import { Page } from '../components/Page';

export function Done({
  info,
  later,
  onLaunch,
  onClose,
}: {
  info: Info;
  /** The components the app fetches itself, with why the installer did not. */
  later: Later[];
  onLaunch: () => Promise<void>;
  onClose: () => void;
}) {
  const [launchError, setLaunchError] = useState('');
  const [details, setDetails] = useState(false);
  const installed = info.mode === 'install';
  const names = later.map(({ id }) => {
    const c = info.components?.find((c) => c.id === id);
    return c ? componentName(c) : id;
  });
  const errors = later.flatMap(({ error }, i) => (error ? [`${names[i]}: ${error}`] : []));

  return (
    <Page
      actions={
        installed ? (
          <>
            <Button label={t('button.close')} labelKey="button.close" tone="neutral" size="key" onClick={onClose} />
            <Button
              label={t('button.start', { name: info.name })}
              labelKey="button.start"
              tone="accent"
              size="key"
              autoFocus
              onClick={() => onLaunch().catch((e: unknown) => setLaunchError(String(e instanceof Error ? e.message : e)))}
            />
          </>
        ) : (
          <Button label={t('button.close')} labelKey="button.close" tone="accent" size="key" autoFocus onClick={onClose} />
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
            {later.map((f, i) => (
              <p key={f.id} className="text-subline text-carbon-textSub">
                {t('done.reason', { component: names[i]!, reason: reasonText(f) })}
              </p>
            ))}
            {errors.length > 0 && (
              <>
                <button
                  type="button"
                  aria-expanded={details}
                  aria-controls="fetch-details"
                  onClick={() => setDetails(!details)}
                  className="flex items-center gap-1 self-start text-subline text-carbon-textMuted hover:text-carbon-text"
                >
                  {t('button.details')}
                  <span className={`flex motion-safe:transition-transform ${details ? 'rotate-180' : ''}`}>
                    <IconExpand />
                  </span>
                </button>
                {details && (
                  <div id="fetch-details" className="flex max-h-16 flex-col gap-1 overflow-y-auto">
                    {/* The errors are in English, so each line keeps its own direction. */}
                    {errors.map((e) => (
                      <p key={e} dir="auto" className="text-subline break-words text-carbon-textMuted select-text">
                        {e}
                      </p>
                    ))}
                  </div>
                )}
              </>
            )}
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
