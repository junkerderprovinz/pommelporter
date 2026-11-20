import { expect, it } from 'vitest';

// Everything under the pointer takes the accent's tint, --hover (GlimStone
// rule 21). The grey steps stay for a field's focus.

const sources = Object.entries(
  import.meta.glob<string>(['./**/*.tsx', './*.css'], { query: '?raw', import: 'default', eager: true }),
);

it('hovers to the accent tint and not to a grey step', () => {
  const grey = sources
    .filter(([path]) => !path.startsWith('./glimstone/'))
    .flatMap(([path, text]) =>
      [...text.matchAll(/hover:bg-carbon-(?:hover|hoverRaised|surface3)\b|:hover[^{]*\{[^}]*var\(--carbon-(?:hover|hover-raised|surface3)\)/g)].map(
        (m) => `${path}: ${m[0]}`,
      ),
    );
  expect(grey).toEqual([]);
});
