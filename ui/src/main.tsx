import { StrictMode, type ReactNode } from 'react';
import { createRoot } from 'react-dom/client';
import '@fontsource-variable/noto-sans';
import '@fontsource-variable/noto-sans-arabic';
import '@fontsource-variable/noto-sans-hebrew';
import '@fontsource-variable/noto-sans-thai';
import './index.css';
import { App } from './App';
import { connect } from './bridge';
import { loadLanguage } from './i18n';
import { applyAccent, applyMotion, applyShape, DEFAULT_MOTION, DEFAULT_SHAPE } from './glimstone/appearance';
import { applyStoredLabelModes } from './glimstone/controls';
import { setGlyphResolver } from './glimstone/react/glyphs';
import {
  IconBack,
  IconCheck,
  IconClose,
  IconDownload,
  IconExternalLink,
  IconFolderAdd,
  IconFolderOpen,
  IconFolderUp,
  IconForward,
  IconPlay,
  IconRefresh,
  IconTrash,
} from './components/icons';

const mirrored = 'rtl:-scale-x-100';

const GLYPHS: Record<string, () => ReactNode> = {
  'button.next': () => <IconForward className={mirrored} />,
  'button.back': () => <IconBack className={mirrored} />,
  'button.cancel': () => <IconClose />,
  'button.close': () => <IconClose />,
  'button.install': () => <IconDownload />,
  'button.uninstall': () => <IconTrash />,
  'button.start': () => <IconPlay />,
  'button.retry': () => <IconRefresh />,
  'button.website': () => <IconExternalLink />,
  'button.browse': () => <IconFolderOpen />,
  'button.up': () => <IconFolderUp />,
  'button.newFolder': () => <IconFolderAdd />,
  'button.create': () => <IconFolderAdd />,
  'button.use': () => <IconCheck />,
};

// The installer has no settings of its own, so every engine starts at the
// house defaults, before the first paint.
applyShape(DEFAULT_SHAPE);
applyMotion(DEFAULT_MOTION);
applyStoredLabelModes();
setGlyphResolver((key) => GLYPHS[key]?.());

const [installer] = await Promise.all([connect(), loadLanguage()]);
const [info, defaults] = await Promise.all([installer.info(), installer.defaults()]);
applyAccent(info.accent);
document.title = info.name;
document.documentElement.classList.toggle('floating', info.frameless);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App installer={installer} info={info} defaults={defaults} />
  </StrictMode>,
);
