import type { ReactNode } from 'react';
import type { Info } from '../bridge';
import { t } from '../i18n';
import { Button } from '../glimstone/react/Button';
import { Logo, Page } from '../components/Page';

function List({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="glim-well flex flex-col gap-2 px-4 py-3">
      <p className="glim-eyebrow">{title}</p>
      <ul className="flex flex-col gap-1.5 text-sm text-carbon-text">{children}</ul>
    </div>
  );
}

function Item({ children }: { children: ReactNode }) {
  return (
    <li className="flex items-baseline gap-2">
      <span className="h-1.5 w-1.5 shrink-0 translate-y-[-2px] rounded-full bg-carbon-textMuted" />
      {children}
    </li>
  );
}

/** The uninstall question: what goes, and that the person's own things stay. */
export function Confirm({ info, onUninstall, onCancel }: { info: Info; onUninstall: () => void; onCancel: () => void }) {
  const linux = info.platform === 'linux';
  const version = info.existing?.version ?? info.version;

  return (
    <Page
      title={t('title.uninstall')}
      actions={
        <>
          <Button label={t('button.cancel')} labelKey="button.cancel" tone="neutral" className="glim-btn-key" autoFocus onClick={onCancel} />
          {/* Not the accent and not red: the page's sentence is what asks. */}
          <Button label={t('button.uninstall')} labelKey="button.uninstall" tone="neutral" className="glim-btn-key" onClick={onUninstall} />
        </>
      }
    >
      <div className="flex items-center gap-4">
        <Logo svg={info.logo} className="h-14 w-14" />
        <div className="flex min-w-0 flex-col gap-0.5">
          <p className="text-title font-semibold text-carbon-text">{info.name}</p>
          <p className="text-body text-carbon-textSub">{t('confirm.lead', { name: info.name, version })}</p>
          {info.existing && (
            <p dir="ltr" className="truncate text-start font-mono text-meta text-carbon-textMuted">
              {info.existing.dir}
            </p>
          )}
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <List title={t('confirm.removed')}>
          <Item>{t('confirm.program')}</Item>
          <Item>{t(linux ? 'options.menu' : 'options.startMenu')}</Item>
          <Item>{t('options.desktop')}</Item>
          {!linux && <Item>{t('confirm.updates')}</Item>}
        </List>
        <List title={t('confirm.kept')}>
          <Item>{t('confirm.data')}</Item>
        </List>
      </div>
    </Page>
  );
}
