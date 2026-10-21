import { describe, expect, it } from 'vitest';
import { en, type Dict } from './locales/en';
import { pickLanguage } from './i18n';
import { refusalText } from './pages/Options';
import { underRoot } from './components/FolderChooser';

const modules = import.meta.glob<Record<string, Dict>>('./locales/*.ts', { eager: true });
const dictionaries = Object.entries(modules).map(([path, module]) => {
  const code = path.slice('./locales/'.length, -'.ts'.length);
  return [code, module[code]!] as const;
});

const placeholders = (s: string) => [...s.matchAll(/\{\w+\}/g)].map((m) => m[0]).sort();

describe('the dictionaries', () => {
  it('cover all 42 languages', () => {
    expect(dictionaries.map(([code]) => code).sort()).toEqual(
      'ar bg ca cs da de el en es et eu fa fi fr gl he hi hr hu id is it ja ko lt lv ms nl no pl pt ro ru sk sl sr sv th tr uk vi zh'.split(' '),
    );
  });

  it.each(dictionaries)('%s has every key and keeps every placeholder', (_code, dict) => {
    expect(Object.keys(dict).sort()).toEqual(Object.keys(en).sort());
    for (const [key, text] of Object.entries(en)) {
      expect(placeholders(dict[key as keyof Dict]), key).toEqual(placeholders(text));
    }
  });

  it.each(dictionaries)('%s has no empty strings and no dashes used as punctuation', (_code, dict) => {
    for (const text of Object.values(dict)) {
      expect(text.trim()).not.toBe('');
      expect(text).not.toMatch(/[\u2013\u2014]/);
    }
  });
});

describe('the language pick', () => {
  it('takes the base of a regional tag', () => {
    expect(pickLanguage('de-AT')).toBe('de');
    expect(pickLanguage('pt-BR')).toBe('pt');
  });

  it('knows the other codes for Norwegian and Hebrew', () => {
    expect(pickLanguage('nb-NO')).toBe('no');
    expect(pickLanguage('iw')).toBe('he');
  });

  it('falls back to English', () => {
    expect(pickLanguage('sw-KE')).toBe('en');
  });
});

describe('a refused folder', () => {
  it('is explained by its code', () => {
    expect(refusalText({ ok: false, code: 'notEmpty', detail: 'x' }, 'KnightLoader')).toBe(en['refusal.notEmpty']);
    expect(refusalText({ ok: false, code: 'fixed' }, 'KnightLoader')).toContain('KnightLoader');
  });

  it('falls back to the platform sentence for a code the window does not know', () => {
    expect(refusalText({ ok: false, code: 'brandNew', detail: 'cannot use it' }, 'KnightLoader')).toBe('cannot use it');
  });
});

describe('a drive root', () => {
  it('holds the paths below it', () => {
    expect(underRoot('C:\\', 'C:\\Program Files')).toBe(true);
    expect(underRoot('D:\\', 'd:\\apps')).toBe(true);
  });

  it('does not hold a sibling that shares its prefix', () => {
    expect(underRoot('/mnt/a', '/mnt/archive')).toBe(false);
  });
});
