import type { ReactNode } from 'react';
import { Card } from '../glimstone/react/Card';

/**
 * One step of the wizard: a card that fills the window, titled with the step's
 * name, and the foot row that answers it. Actions stand at the end of the
 * foot with the forward one last; `start` holds a link out of the window.
 */
export function Page({
  title,
  children,
  actions,
  start,
}: {
  title: string;
  children: ReactNode;
  actions?: ReactNode;
  start?: ReactNode;
}) {
  return (
    <div className="flex h-full flex-col gap-4 px-5 pt-6 pb-4">
      {/* A one-row grid stretches the card to the height that is left. */}
      <main className="glim-page-enter grid min-h-0 flex-1 grid-rows-1">
        <Card title={title}>{children}</Card>
      </main>
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
