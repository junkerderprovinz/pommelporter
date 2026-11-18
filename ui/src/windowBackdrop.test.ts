import { expect, it } from 'vitest';

// A window that holds a form stays open on a press beside it (GlimStone 4.5.6),
// so what was typed there is not lost to a stray click. Cancel and Escape are
// its ways out.

const sources = Object.entries(
  import.meta.glob<string>('./**/*.tsx', { query: '?raw', import: 'default', eager: true }),
);

it('leaves every window that holds a form open on a press beside it', () => {
  const closing = sources
    .filter(([, text]) => /<form\b/.test(text))
    .filter(([, text]) => /glim-modal-backdrop[^>]*onClick=/.test(text))
    .map(([path]) => path);
  expect(closing).toEqual([]);
});
