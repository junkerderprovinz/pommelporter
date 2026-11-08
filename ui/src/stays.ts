// What an uninstall leaves on the computer: in the app's own words where it names them, in the language the window
// speaks or else in English, and otherwise in the window's.
import type { Info } from './bridge';
import { language, t } from './i18n';

export function stays(info: Pick<Info, 'stays'>, part: 'item' | 'note', lang = language()): string {
  const own = info.stays?.[lang] ?? info.stays?.en;
  return own?.[part] || t(part === 'item' ? 'confirm.data' : 'done.kept');
}
