import type { CSSProperties, ReactNode } from 'react';
import type { Info } from '../bridge';
import { t } from '../i18n';
import { IconClose, IconMinimise } from './icons';
import { Logo } from './Page';

// Wails moves the window by any element marked draggable, so the buttons on
// the bar opt out again.
const drag = { '--wails-draggable': 'drag' } as CSSProperties;
const noDrag = { '--wails-draggable': 'no-drag' } as CSSProperties;

/**
 * The window's own title bar where the system draws none: the app's logo and
 * name to move the window by, and the two window buttons. Close is held back
 * while the installer works, as the window would only refuse it.
 */
export function TitleBar({
  info,
  closable,
  onMinimise,
  onClose,
}: {
  info: Info;
  closable: boolean;
  onMinimise: () => void;
  onClose: () => void;
}) {
  return (
    <header className="flex h-9 shrink-0 items-center gap-2 ps-4 select-none" style={drag}>
      <Logo svg={info.logo} className="h-4 w-4 shrink-0" />
      <span className="truncate text-xs text-carbon-textMuted">{info.name}</span>
      <span className="flex-1" />
      <WindowButton label={t('button.minimise')} onClick={onMinimise}>
        <IconMinimise />
      </WindowButton>
      <WindowButton label={t('button.close')} disabled={!closable} onClick={onClose}>
        <IconClose />
      </WindowButton>
    </header>
  );
}

function WindowButton({
  label,
  disabled = false,
  onClick,
  children,
}: {
  label: string;
  disabled?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      disabled={disabled}
      onClick={onClick}
      style={noDrag}
      className="flex h-9 w-11 items-center justify-center text-carbon-textMuted transition-colors
        hover:bg-carbon-surface3 hover:text-carbon-text disabled:opacity-40
        disabled:hover:bg-transparent disabled:hover:text-carbon-textMuted"
    >
      {children}
    </button>
  );
}
