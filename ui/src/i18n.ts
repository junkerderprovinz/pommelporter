// The window speaks the system's language, read from navigator.language, and
// English where it has no dictionary. Only that dictionary is loaded.
import { en, type Dict, type Key } from './locales/en';

export type { Key };

const RTL = new Set(['ar', 'fa', 'he']);

// Browsers report some languages under a second code.
const ALIASES: Record<string, string> = { nb: 'no', nn: 'no', iw: 'he', in: 'id' };

// Each locale module exports its dictionary under its own code.
const dictionaries = import.meta.glob<Record<string, Dict>>(['./locales/*.ts', '!./locales/en.ts']);

const available = new Set(['en', ...Object.keys(dictionaries).map((p) => p.slice('./locales/'.length, -'.ts'.length))]);

/** The dictionary code for a BCP 47 tag such as "de-AT", or "en". */
export function pickLanguage(tag: string): string {
  const base = tag.toLowerCase().split('-')[0] ?? '';
  const code = ALIASES[base] ?? base;
  return available.has(code) ? code : 'en';
}

let dict: Dict = en;
let current = 'en';

/** The code of the language the window speaks. */
export function language(): string {
  return current;
}

/** Switches to the language of tag, loading its dictionary the first time, and sets lang and dir on the page. */
export async function setLanguage(tag: string): Promise<void> {
  const code = pickLanguage(tag);
  dict = code === 'en' ? en : (await dictionaries[`./locales/${code}.ts`]!())[code]!;
  current = code;
  document.documentElement.lang = code;
  document.documentElement.dir = RTL.has(code) ? 'rtl' : 'ltr';
}

export function t(key: Key, vars: Record<string, string> = {}): string {
  return dict[key].replace(/\{(\w+)\}/g, (whole, name: string) => vars[name] ?? whole);
}
