import type { InputHTMLAttributes } from 'react';

/**
 * A one-line field in the GlimStone form: borderless and filled, with focus
 * shown as a step up the surface ramp. Paths keep `dir="ltr"` at the call
 * site, so a right-to-left page does not reorder their separators.
 */
export function TextField({ className = '', ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      spellCheck={false}
      autoComplete="off"
      {...props}
      className={`h-[var(--btn-h)] min-w-0 flex-1 rounded-control bg-carbon-surface2 px-3 text-start text-sm text-carbon-text
        outline-none placeholder:text-carbon-textMuted focus:bg-carbon-surface3 ${className}`}
    />
  );
}
