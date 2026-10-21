import type { ReactNode } from 'react';

/**
 * One step of the wizard inside the window's card: its content under the logo
 * and the foot row that answers it. Actions stand at the end of the foot with
 * the forward one last; `start` holds a link out of the window.
 */
export function Page({
  children,
  actions,
  start,
}: {
  children: ReactNode;
  actions?: ReactNode;
  start?: ReactNode;
}) {
  return (
    <div className="flex h-full flex-col gap-4 px-6 pt-2 pb-5">
      <main className="glim-page-enter flex min-h-0 flex-1 flex-col">{children}</main>
      <footer className="flex min-h-[var(--btn-h-key)] items-center gap-3">
        {start}
        <span className="flex-1" />
        {actions}
      </footer>
    </div>
  );
}

/** The app's logo from Info().logo, drawn as an image so its own styles stay inside it. */
export function Logo({ svg, className }: { svg: string; className: string }) {
  return (
    <img
      src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`}
      alt=""
      draggable={false}
      className={`shrink-0 select-none ${className}`}
    />
  );
}
