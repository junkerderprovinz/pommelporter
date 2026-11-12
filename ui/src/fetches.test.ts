import { describe, expect, it } from 'vitest';
import { en } from './locales/en';
import { amountText, componentName, nameList, reasonText, sizeText } from './fetches';
import { WAITING } from './run';

describe('a component', () => {
  const component = { id: 'runtime', name: { en: 'Runtime', de: 'Laufzeitumgebung' } };

  it('is named in the language of the window', () => {
    expect(componentName(component, 'de')).toBe('Laufzeitumgebung');
  });

  it('falls back to English, then to its id', () => {
    expect(componentName(component, 'fi')).toBe('Runtime');
    expect(componentName({ id: 'runtime', name: {} }, 'fi')).toBe('runtime');
  });
});

describe('a download', () => {
  const running = (done: number, total: number) => ({ ...WAITING, state: 'running' as const, done, total });

  it('says how much of it has arrived', () => {
    expect(sizeText(running(42_000_000, 180_000_000), 'en')).toBe('42 MB of 180 MB');
  });

  it('keeps a decimal for a small one', () => {
    expect(sizeText(running(1_250_000, 3_200_000), 'en')).toBe('1.3 MB of 3.2 MB');
  });

  it('shows only what arrived while the size is unknown, and nothing before the first byte', () => {
    expect(sizeText(running(5_000_000, -1), 'en')).toBe('5 MB');
    expect(sizeText(running(0, 100), 'en')).toBe('');
  });
});

describe('a component the program fetches', () => {
  const running = (fraction: number | null) => ({ ...WAITING, state: 'running' as const, fraction });

  it('says how far the program has come', () => {
    expect(amountText(running(0.4), 'en')).toBe('40%');
    expect(amountText(running(1), 'de')).toBe('100 %');
  });

  it('says nothing until the program names a share', () => {
    expect(amountText(running(null), 'en')).toBe('');
  });

  it('shares the words of a download otherwise', () => {
    expect(amountText({ ...WAITING, state: 'running', done: 5_000_000, total: -1 }, 'en')).toBe('5 MB');
  });
});

describe('a component left for later', () => {
  it('is explained in plain words for the common causes', () => {
    expect(reasonText({ state: 'failed', reason: 'network' })).toBe(en['reason.network']);
    expect(reasonText({ state: 'failed', reason: 'server' })).toBe(en['reason.server']);
    expect(reasonText({ state: 'failed', reason: 'checksum' })).toBe(en['reason.checksum']);
  });

  it('gets a general sentence for any other cause, and says when it was skipped', () => {
    expect(reasonText({ state: 'failed', reason: 'other' })).toBe(en['reason.other']);
    expect(reasonText({ state: 'failed', reason: '' })).toBe(en['reason.other']);
    expect(reasonText({ state: 'skipped', reason: '' })).toBe(en['reason.skipped']);
  });
});

describe('the names left for later', () => {
  it('are listed the way the language lists things', () => {
    expect(nameList(['A', 'B', 'C'], 'en')).toBe('A, B, and C');
    expect(nameList(['A', 'B'], 'de')).toBe('A und B');
  });
});
