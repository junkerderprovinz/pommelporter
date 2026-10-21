import { useRef, useState } from 'react';
import { DropdownListbox } from '../glimstone/react/DropdownListbox';
import { flagURL, LANGUAGES } from '../languages';

function Flag({ flag }: { flag: string }) {
  const src = flagURL(flag);
  return src ? <img src={src} alt="" className="h-[1em] w-[1.25em] shrink-0 rounded-[2px] object-cover" /> : null;
}

/** The language the installer speaks and the app first starts in. */
export function LanguagePicker({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (code: string) => void;
}) {
  const trigger = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const current = LANGUAGES.find((l) => l.code === value) ?? LANGUAGES[0]!;

  return (
    <div className="relative w-56">
      <button
        ref={trigger}
        type="button"
        aria-label={label}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="flex h-[var(--btn-h-key)] w-full items-center gap-2.5 bg-carbon-surface3 ps-3 pe-8 text-start text-body
          text-carbon-text outline-none transition hover:bg-carbon-hoverRaised focus-visible:brightness-125"
        style={{ borderRadius: 'var(--radius-control)' }}
      >
        <Flag flag={current.flag} />
        <span className="min-w-0 flex-1 truncate">{current.label}</span>
      </button>
      {/* Outside the button, so its flex row reserves no space for the chevron. */}
      <svg
        aria-hidden
        width="10"
        height="10"
        viewBox="0 0 10 10"
        className="pointer-events-none absolute end-3 top-1/2 -translate-y-1/2 text-carbon-textMuted"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
      >
        <path d="M2 4l3 3 3-3" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
      <DropdownListbox open={open} onClose={() => setOpen(false)} triggerRef={trigger} label={label}>
        {LANGUAGES.map((l) => (
          <button
            key={l.code}
            type="button"
            role="option"
            aria-selected={l.code === value}
            onClick={() => {
              onChange(l.code);
              setOpen(false);
            }}
            className={`flex w-full items-center gap-2.5 px-3 py-2 text-start text-body transition-colors ${
              l.code === value ? 'bg-carbon-surface3 text-carbon-text' : 'text-carbon-textSub hover:bg-carbon-hover hover:text-carbon-text'
            }`}
          >
            <Flag flag={l.flag} />
            <span className="min-w-0 truncate">{l.label}</span>
          </button>
        ))}
      </DropdownListbox>
    </div>
  );
}
