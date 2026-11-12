import { describe, expect, it } from 'vitest';
import { componentName, nameList, sizeText } from './fetches';

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
  const running = (done: number, total: number) => ({ state: 'running' as const, done, total, error: '' });

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

describe('the names left for later', () => {
  it('are listed the way the language lists things', () => {
    expect(nameList(['A', 'B', 'C'], 'en')).toBe('A, B, and C');
    expect(nameList(['A', 'B'], 'de')).toBe('A und B');
  });
});
