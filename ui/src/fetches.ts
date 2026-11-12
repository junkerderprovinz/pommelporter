// The words for the components an install fetches: their names in the app's
// own words, and how much of a download has arrived.
import type { Component } from './bridge';
import type { Fetch } from './run';
import { language, t } from './i18n';

/** A component's name in the language the window speaks, else in English, else its id. */
export function componentName(component: Component, lang = language()): string {
  return component.name[lang] ?? component.name.en ?? component.id;
}

const MB = 1_000_000;

function megabytes(bytes: number, lang: string): string {
  const mb = bytes / MB;
  return new Intl.NumberFormat(lang, {
    style: 'unit',
    unit: 'megabyte',
    maximumFractionDigits: mb < 10 ? 1 : 0,
  }).format(mb);
}

/** How much of a running download has arrived, such as "42 of 180 MB"; empty before the first byte. */
export function sizeText(fetch: Fetch, lang = language()): string {
  if (fetch.done <= 0) return '';
  if (fetch.total <= 0) return megabytes(fetch.done, lang);
  return t('component.size', { done: megabytes(fetch.done, lang), total: megabytes(fetch.total, lang) });
}

/** The names joined the way the language lists things, such as "A, B and C". */
export function nameList(names: string[], lang = language()): string {
  return new Intl.ListFormat(lang, { type: 'conjunction' }).format(names);
}
