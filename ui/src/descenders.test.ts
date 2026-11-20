import { expect, it } from 'vitest';
import css from './index.css?raw';

// A button's label hides its overflow so a long one ends in an ellipsis. Its
// line box clears Latin descenders, and the tail of an Arabic ع or ي reaches
// about 1.5px further.

it("keeps the descenders of a button's label inside its clip", () => {
  const rule = css.match(/\.glim-btn-label\s*\{([^}]*)\}/)?.[1] ?? '';
  expect(rule).toMatch(/padding-block:\s*0?\.15em;/);
  expect(rule).toMatch(/margin-block:\s*-0?\.15em;/);
});
