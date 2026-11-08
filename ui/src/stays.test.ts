import { describe, expect, it } from 'vitest';
import { stays } from './stays';

const own = { de: { item: 'Deine Einstellungen', note: 'Deine Einstellungen sind noch da.' }, en: { item: 'Your settings', note: 'Your settings are still there.' } };

describe('what an uninstall leaves', () => {
  it("is the app's own, in the language of the window", () => {
    expect(stays({ stays: own }, 'item', 'de')).toBe('Deine Einstellungen');
    expect(stays({ stays: own }, 'note', 'de')).toBe('Deine Einstellungen sind noch da.');
  });

  it('falls back to English for a language the app lacks', () => {
    expect(stays({ stays: own }, 'note', 'fi')).toBe('Your settings are still there.');
  });

  it("keeps the window's own words for an app that names nothing", async () => {
    const { setLanguage } = await import('./i18n');
    await setLanguage('en');
    expect(stays({ stays: null }, 'item', 'en')).toBe('Your downloads, settings and accounts');
    expect(stays({}, 'note', 'en')).toBe('Your downloads, settings and accounts are still there.');
  });
});
