// The folder chooser behind the install folder's Browse button: the path as
// text, the drives as a row to pick from, the folders of the listed one, and
// a way to make a new folder. The text box is the answer; the list only helps
// to fill it.
import { useEffect, useId, useState } from 'react';
import { createPortal } from 'react-dom';
import type { Folder, Installer } from '../bridge';
import { t } from '../i18n';
import { Badge } from '../glimstone/react/Badge';
import { Button } from '../glimstone/react/Button';
import { IconFolder } from './icons';
import { TextField } from './TextField';

/** Whether path is root itself or lies below it, on either kind of separator. */
export function underRoot(root: string, path: string): boolean {
  const base = root.replace(/[\\/]+$/, '');
  if (!path.toLowerCase().startsWith(base.toLowerCase())) return false;
  const next = path.charAt(base.length);
  return next === '' || next === '\\' || next === '/';
}

export function FolderChooser({
  installer,
  value,
  suggestedName,
  onPick,
  onClose,
}: {
  installer: Installer;
  value: string;
  /** What a new folder is called until the person types something else. */
  suggestedName: string;
  onPick: (dir: string) => void;
  onClose: () => void;
}) {
  const [text, setText] = useState(value);
  const [query, setQuery] = useState(value);
  const [folder, setFolder] = useState<Folder | null>(null);
  const [error, setError] = useState('');
  const [naming, setNaming] = useState(false);
  const [name, setName] = useState(suggestedName);
  const [nameError, setNameError] = useState('');
  const titleId = useId();

  useEffect(() => {
    let live = true;
    installer
      .listDir(query)
      .then((f) => {
        if (!live) return;
        setFolder(f);
        setError('');
      })
      // The last good listing stays, so there is a way back.
      .catch((e: unknown) => live && setError(String(e instanceof Error ? e.message : e)));
    return () => {
      live = false;
    };
  }, [installer, query]);

  // Typing browses too, once it pauses.
  useEffect(() => {
    if (text === query) return;
    const id = setTimeout(() => setQuery(text), 300);
    return () => clearTimeout(id);
  }, [text, query]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !naming) onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [naming, onClose]);

  function go(path: string) {
    setText(path);
    setQuery(path);
  }

  async function create() {
    if (!folder || name.trim() === '') return;
    try {
      go(await installer.makeDir(folder.path, name.trim()));
      setNaming(false);
      setNameError('');
    } catch (e) {
      setNameError(String(e instanceof Error ? e.message : e));
    }
  }

  const roots = folder?.roots ?? [];
  // The longest matching root, since one root can sit inside another.
  const activeRoot = roots.filter((r) => underRoot(r, text)).sort((a, b) => b.length - a.length)[0];

  return createPortal(
    <div
      className="glim-modal-backdrop fixed inset-0 z-50 flex items-stretch justify-center p-4"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="glim-modal-card relative mt-2 flex w-full flex-col gap-3 rounded-card bg-carbon-surface p-5 pt-6 shadow-2xl"
      >
        <h2 id={titleId} className="flex items-center">
          <Badge tone="heading" size="heading" insetStart={5}>
            {t('title.chooser')}
          </Badge>
        </h2>

        <form
          className="flex items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            setQuery(text);
          }}
        >
          <Button
            label={t('button.up')}
            labelKey="button.up"
            variant="icon"
            tone="subtle"
            disabled={!folder?.parent}
            onClick={() => folder?.parent && go(folder.parent)}
          />
          <TextField
            dir="ltr"
            autoFocus
            value={text}
            aria-label={t('chooser.path')}
            onChange={(e) => setText(e.target.value)}
          />
        </form>

        {roots.length > 1 && (
          <div role="group" aria-label={t('chooser.drives')} className="flex flex-wrap gap-1.5" dir="ltr">
            {roots.map((root) => {
              const on = root === activeRoot;
              return (
                <button
                  key={root}
                  type="button"
                  aria-pressed={on}
                  onClick={() => go(root)}
                  className={`h-7 min-w-14 rounded-pill px-3 text-sm font-medium transition-colors ${
                    on ? 'bg-accent text-accentContrast' : 'bg-carbon-surface2 text-carbon-textSub hover:bg-carbon-surface3 hover:text-carbon-text'
                  }`}
                >
                  {root}
                </button>
              );
            })}
          </div>
        )}

        {/* Paths read left to right in every language, so the list does too. */}
        <div dir="ltr" className="glim-well min-h-24 flex-1 overflow-y-auto py-1">
          {folder === null && !error && (
            <p className="px-3 py-6 text-center text-subline text-carbon-textMuted">{t('chooser.loading')}</p>
          )}
          {folder?.entries.map((entry) => (
            <button
              key={entry.path}
              type="button"
              onClick={() => go(entry.path)}
              className="flex w-full items-center gap-2 px-3 py-1.5 text-start text-sm text-carbon-text transition-colors hover:bg-carbon-surface3"
            >
              <IconFolder className="shrink-0 text-carbon-textMuted" />
              <span className="truncate">{entry.name}</span>
            </button>
          ))}
          {folder && folder.entries.length === 0 && (
            <p className="px-3 py-6 text-center text-subline text-carbon-textMuted">{t('chooser.empty')}</p>
          )}
        </div>

        {naming ? (
          <form
            className="flex items-center gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              void create();
            }}
          >
            <TextField
              autoFocus
              dir="auto"
              value={name}
              aria-label={t('chooser.newName')}
              placeholder={t('chooser.newName')}
              onChange={(e) => {
                setName(e.target.value);
                setNameError('');
              }}
              onKeyDown={(e) => {
                // Escape gives up the name, not the whole chooser.
                if (e.key === 'Escape') setNaming(false);
              }}
            />
            <Button label={t('button.create')} labelKey="button.create" type="submit" tone="neutral" disabled={name.trim() === ''} />
          </form>
        ) : (
          <div>
            <Button label={t('button.newFolder')} labelKey="button.newFolder" tone="neutral" disabled={!folder} onClick={() => setNaming(true)} />
          </div>
        )}

        {(nameError || error) && (
          // The sentence comes from the system in English, so it keeps its own direction.
          <p role="alert" dir="auto" className="text-subline text-statusFail">
            {nameError || error}
          </p>
        )}

        <div className="flex items-center justify-end gap-3 pt-1">
          <Button label={t('button.cancel')} labelKey="button.cancel" tone="neutral" className="glim-btn-key" onClick={onClose} />
          <Button
            label={t('button.use')}
            labelKey="button.use"
            tone="accent"
            className="glim-btn-key"
            disabled={text.trim() === ''}
            onClick={() => onPick(text.trim())}
          />
        </div>
      </div>
    </div>,
    document.body,
  );
}
