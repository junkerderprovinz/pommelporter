import type { CSSProperties, ReactNode } from 'react';
import type { Info } from '../bridge';
import { t } from '../i18n';
import { IconClose, IconMinimise } from './icons';
import { Logo } from './Page';

// Wails moves the window by any element marked draggable, so the buttons opt
// out again.
const drag = { '--wails-draggable': 'drag' } as CSSProperties;
const noDrag = { '--wails-draggable': 'no-drag' } as CSSProperties;

/**
 * The whole window as one card with the app's logo on its top edge. Where the
 * window has no frame of its own, the card floats in a clear window with room
 * above it for the logo and around it for the shadow, carries the window's two
 * buttons, and is what the window is dragged by. Close is held back while the
 * installer works, as the window would only refuse it.
 */
export function WindowCard({
  info,
  closable,
  onMinimise,
  onClose,
  children,
}: {
  info: Info;
  closable: boolean;
  onMinimise: () => void;
  onClose: () => void;
  children: ReactNode;
}) {
  if (!info.frameless) {
    return (
      <div className="flex h-full flex-col bg-carbon-surface">
        <div className="flex shrink-0 justify-center pt-5">
          <Logo svg={info.logo} className="h-20 max-w-[50%] object-contain" />
        </div>
        <div className="min-h-0 flex-1">{children}</div>
      </div>
    );
  }
  return (
    <div className="relative h-full px-3 pt-14 pb-3">
      <div className="glim-card relative flex h-full flex-col">
        <div className="absolute inset-x-0 top-0 h-16" style={drag} />
        <div className="absolute end-1 top-1 flex">
          <WindowButton label={t('button.minimise')} onClick={onMinimise}>
            <IconMinimise />
          </WindowButton>
          <WindowButton label={t('button.close')} disabled={!closable} onClick={onClose}>
            <IconClose />
          </WindowButton>
        </div>
        <div className="min-h-0 flex-1 pt-16">{children}</div>
      </div>
      <div className="absolute start-1/2 top-0 flex max-w-[60%] -translate-x-1/2 rtl:translate-x-1/2" style={drag}>
        <Logo svg={info.logo} className="h-28 max-w-full object-contain drop-shadow-lg" />
      </div>
    </div>
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
      className="flex h-9 w-10 items-center justify-center rounded-control text-carbon-textMuted transition-colors
        hover:bg-carbon-surface3 hover:text-carbon-text disabled:opacity-40
        disabled:hover:bg-transparent disabled:hover:text-carbon-textMuted"
    >
      {children}
    </button>
  );
}
