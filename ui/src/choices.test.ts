import { describe, expect, it } from 'vitest';
import { choiceLabel } from './pages/Options';

describe('an app choice', () => {
  const choice = { id: 'folders', label: { de: 'Als Standard-Dateimanager festlegen', en: 'Make it the default file manager' } };

  it('reads in the language of the window', () => {
    expect(choiceLabel(choice, 'de')).toBe('Als Standard-Dateimanager festlegen');
  });

  it('falls back to English, then to its id', () => {
    expect(choiceLabel(choice, 'fi')).toBe('Make it the default file manager');
    expect(choiceLabel({ id: 'folders', label: {} }, 'fi')).toBe('folders');
  });
});
