import type { Info } from '../bridge';
import { t } from '../i18n';
import { Button } from '../glimstone/react/Button';
import { Page } from '../components/Page';

function lead(info: Info): string {
  const { name, version, existing } = info;
  if (existing && existing.version === version) return t('welcome.repair', { name, version });
  if (existing) return t('welcome.update', { name, version, old: existing.version });
  return t(info.platform === 'linux' ? 'welcome.linux' : 'welcome.windows', { name });
}

export function Welcome({
  info,
  onNext,
  onCancel,
  onWebsite,
}: {
  info: Info;
  onNext: () => void;
  onCancel: () => void;
  onWebsite: () => void;
}) {
  return (
    <Page
      start={
        info.homepage && (
          <Button label={t('button.website')} labelKey="button.website" tone="subtle" className="glim-btn-key" onClick={onWebsite} />
        )
      }
      actions={
        <>
          <Button label={t('button.cancel')} labelKey="button.cancel" tone="neutral" className="glim-btn-key" onClick={onCancel} />
          <Button label={t('button.next')} labelKey="button.next" tone="accent" className="glim-btn-key" autoFocus onClick={onNext} />
        </>
      }
    >
      <div className="flex flex-1 flex-col items-center justify-center gap-4 text-center">
        <div className="flex flex-col gap-1">
          <p className="text-status font-semibold text-carbon-text">{info.name}</p>
          <p className="text-subline text-carbon-textSub">
            {t('welcome.byline', { version: info.version, publisher: info.publisher })}
          </p>
        </div>
        <p className="text-body text-carbon-textSub">{lead(info)}</p>
      </div>
    </Page>
  );
}
