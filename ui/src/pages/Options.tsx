import { useEffect, useState } from 'react';
import type { DirCheck, Info, Installer, Options as Choices } from '../bridge';
import { t, type Key } from '../i18n';
import { Button } from '../glimstone/react/Button';
import { InfoBubble } from '../glimstone/react/InfoBubble';
import { Toggle } from '../glimstone/react/Toggle';
import { FolderChooser } from '../components/FolderChooser';
import { IconWarning } from '../components/icons';
import { Page } from '../components/Page';
import { TextField } from '../components/TextField';

const REFUSALS: Record<string, Key> = {
  relative: 'refusal.relative',
  notLocal: 'refusal.notLocal',
  noAcl: 'refusal.noAcl',
  notEmpty: 'refusal.notEmpty',
  unsafeParent: 'refusal.unsafeParent',
  link: 'refusal.link',
  file: 'refusal.file',
  fixed: 'refusal.fixed',
};

/** Why a folder was refused, in words a person can act on; the platform's own sentence for a code this window does not know. */
export function refusalText(check: DirCheck, name: string): string {
  const key = check.code ? REFUSALS[check.code] : undefined;
  return key ? t(key, { name }) : (check.detail ?? '');
}

export function Options({
  installer,
  info,
  choices,
  onChange,
  onInstall,
  onBack,
  onCancel,
}: {
  installer: Installer;
  info: Info;
  choices: Choices;
  onChange: (next: Choices) => void;
  onInstall: () => void;
  onBack: () => void;
  onCancel: () => void;
}) {
  const [checked, setChecked] = useState<{ dir: string; result: DirCheck } | null>(null);
  const [choosing, setChoosing] = useState(false);
  const linux = info.platform === 'linux';
  const dir = choices.dir;

  // Checked as the person types, once the typing pauses.
  useEffect(() => {
    let live = true;
    const id = setTimeout(() => {
      installer.checkDir(dir).then((result) => live && setChecked({ dir, result }));
    }, 250);
    return () => {
      live = false;
      clearTimeout(id);
    };
  }, [installer, dir]);

  const current = checked?.dir === dir ? checked.result : null;
  const refusal = current && !current.ok ? refusalText(current, info.name) : '';

  return (
    <Page
      title={t('title.options')}
      actions={
        <>
          <Button label={t('button.cancel')} labelKey="button.cancel" tone="neutral" className="glim-btn-key" onClick={onCancel} />
          <Button label={t('button.back')} labelKey="button.back" tone="neutral" className="glim-btn-key" onClick={onBack} />
          <Button
            label={t('button.install')}
            labelKey="button.install"
            tone="accent"
            className="glim-btn-key"
            disabled={!current?.ok}
            onClick={onInstall}
          />
        </>
      }
    >
      <div className="flex flex-col gap-5">
        {!linux && (
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-4">
              <span className="flex shrink-0 items-center gap-1.5 text-sm text-carbon-text">
                {t('options.folder')}
                <InfoBubble tip={t('options.folderHint', { name: info.name })} />
              </span>
              <span className="flex min-w-0 flex-1 items-center gap-2">
                <TextField
                  dir="ltr"
                  value={dir}
                  aria-label={t('options.folder')}
                  aria-invalid={refusal !== ''}
                  onChange={(e) => onChange({ ...choices, dir: e.target.value })}
                />
                <Button label={t('button.browse')} labelKey="button.browse" tone="neutral" onClick={() => setChoosing(true)} />
              </span>
            </div>
            {refusal && (
              <p role="alert" className="flex items-start gap-2 text-subline text-statusWarn">
                <IconWarning className="mt-0.5 shrink-0" width={14} height={14} />
                <span>{refusal}</span>
              </p>
            )}
          </div>
        )}
        <Toggle
          className="justify-between"
          label={t(linux ? 'options.menu' : 'options.startMenu')}
          checked={choices.startMenu}
          onChange={(startMenu) => onChange({ ...choices, startMenu })}
        />
        <Toggle
          className="justify-between"
          label={t('options.desktop')}
          checked={choices.desktop}
          onChange={(desktop) => onChange({ ...choices, desktop })}
        />
      </div>
      {choosing && (
        <FolderChooser
          installer={installer}
          value={dir}
          suggestedName={info.name}
          onClose={() => setChoosing(false)}
          onPick={(picked) => {
            onChange({ ...choices, dir: picked });
            setChoosing(false);
          }}
        />
      )}
    </Page>
  );
}
