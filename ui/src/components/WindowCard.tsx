import type { CSSProperties, ReactNode } from 'react';
import type { Info } from '../bridge';
import { t } from '../i18n';
import { IconClose, IconWindowMinimise } from '../glimstone/react/glyphs';
import { IconTipButton } from '../glimstone/react/IconTipButton';
import { Logo } from './Page';

// Wails moves the window by any element marked draggable, so the buttons opt
// out again.
const drag = { '--wails-draggable': 'drag' } as CSSProperties;
const noDrag = { '--wails-draggable': 'no-drag' } as CSSProperties;

/**
 * The whole window as one card, with the app's logo on its tile at the top.
 * Where the window has no frame of its own, the card floats in a clear window
 * with room around it for the shadow, carries the window's two buttons, and is
 * what the window is dragged by. Close is held back while the installer works,
 * as the window would only refuse it.
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
  const content = (
    <>
      <div className="flex shrink-0 justify-center pt-7" style={drag}>
        <AppTile svg={info.logo} />
      </div>
      <div className="min-h-0 flex-1">{children}</div>
    </>
  );
  if (!info.frameless) {
    return <div className="flex h-full flex-col bg-carbon-surface">{content}</div>;
  }
  return (
    <div className="h-full p-3">
      <div className="glim-card relative flex h-full flex-col">
        <div className="absolute end-1 top-1 z-10 flex h-10">
          <WindowButton label={t('button.minimise')} onPress={onMinimise}>
            <IconWindowMinimise />
          </WindowButton>
          <WindowButton label={t('button.close')} disabled={!closable} onPress={onClose}>
            <IconClose />
          </WindowButton>
        </div>
        {content}
      </div>
    </div>
  );
}

/**
 * The logo on the app's tile, lit from the left as on its icon, so it stands
 * on the card like on a pedestal. The tile follows the colour mode.
 */
function AppTile({ svg }: { svg: string }) {
  return (
    <div
      className="relative flex size-36 items-center justify-center rounded-[2rem]"
      style={{
        background: 'linear-gradient(90deg, var(--tile-lit) 50%, var(--tile-shade) 50%)',
        boxShadow: 'var(--elevation), var(--hairline)',
      }}
    >
      <Logo svg={svg} className="h-32 max-w-32 object-contain" />
    </div>
  );
}

// The window buttons of GlimStone's WindowBar, standing in the card's corner
// instead of on a bar, since the card is the window.
function WindowButton({
  label,
  disabled = false,
  onPress,
  children,
}: {
  label: string;
  disabled?: boolean;
  onPress: () => void;
  children: ReactNode;
}) {
  return (
    <IconTipButton tip={label} disabled={disabled} onClick={onPress} style={noDrag} className="glim-window-button">
      <span className="glim-window-square">{children}</span>
    </IconTipButton>
  );
}
