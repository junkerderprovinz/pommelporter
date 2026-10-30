import { describe, expect, it } from 'vitest';

// A window's way out and its other answers are filled buttons (GlimStone rules
// 3 and 15). Every tone of GlimStone's Button has a fill, so a foot holds
// nothing but Button: a bare <button> or a link there, or a Button stripped of
// its ground, reads as a word beside the button it answers with.

const sources = Object.entries(
  import.meta.glob<string>('./**/*.tsx', { query: '?raw', import: 'default', eager: true }),
);

/** The end of the brace that opens at `from`. */
function closingBrace(text: string, from: number): number {
  let depth = 0;
  for (let i = from; i < text.length; i++) {
    if (text[i] === '{') depth++;
    else if (text[i] === '}' && --depth === 0) return i + 1;
  }
  return text.length;
}

/** Each foot with where it starts: a Page's `actions` and `start`, and every `<footer>`. */
function feet(text: string): [string, number][] {
  const out: [string, number][] = [];
  for (const m of text.matchAll(/\b(actions|start)=\{/g)) {
    out.push([text.slice(m.index, closingBrace(text, m.index + m[0].length - 1)), m.index]);
  }
  for (const m of text.matchAll(/<footer\b[\s\S]*?<\/footer>/g)) {
    out.push([m[0], m.index]);
  }
  return out;
}

const UNFILLED = [/<button\b/, /<a\b/, /\bbg-(transparent|none)\b/];

describe('window feet', () => {
  it('are found in the source', () => {
    expect(sources.flatMap(([, text]) => feet(text)).length).toBeGreaterThan(5);
  });

  it('fill every button they hold', () => {
    const hits: string[] = [];
    for (const [path, text] of sources) {
      for (const [foot, at] of feet(text)) {
        for (const shape of UNFILLED) {
          const m = shape.exec(foot);
          if (!m) continue;
          const line = text.slice(0, at + m.index).split('\n').length;
          hits.push(`${path}:${line} ${m[0]}`);
        }
      }
    }
    expect(hits).toEqual([]);
  });
});
