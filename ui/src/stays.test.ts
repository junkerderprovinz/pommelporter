import { describe, expect, it } from 'vitest';
import { t } from './i18n';
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

  it("keeps the window's own words for an app that names nothing", () => {
    expect(stays({ stays: null }, 'item', 'en')).toBe(t('confirm.data'));
    expect(stays({}, 'note', 'en')).toBe(t('done.kept'));
  });
});
