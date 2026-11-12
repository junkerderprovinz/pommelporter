// The window's way to the Installer service in window.go. Inside the installer
// Wails serves its runtime at /wails/runtime.js; `npm run dev` in a plain
// browser gets the stand-in from mock.ts instead, which the build leaves out.

/** A copy that is already installed, as setup.Existing. */
export interface Existing {
  version: string;
  dir: string;
}

/** What the window shows before anything happens, as pommelporter.Info. */
export interface Info {
  name: string;
  version: string;
  publisher: string;
  homepage: string;
  accent: string;
  /** The app's logo as SVG markup. */
  logo: string;
  mode: 'install' | 'uninstall';
  platform: string;
  existing: Existing | null;
  /** Whether the window draws its own title bar, as on Windows. */
  frameless: boolean;
  /** The app's own ticks for this platform. */
  choices: Choice[] | null;
  /** What an uninstall leaves, by language code; null for the window's own words. */
  stays?: Record<string, Stay> | null;
  /** What the installer fetches once the program is in place, for this platform. */
  components?: Component[] | null;
}

/** Something the installer fetches after the program, as setup.Component. */
export interface Component {
  id: string;
  /** The name the window shows, by language code. */
  name: Record<string, string>;
}

/**
 * How far one component has come. "present" was there already, and "failed"
 * and "skipped" are left for the app to fetch itself.
 */
export type FetchState = 'running' | 'done' | 'present' | 'failed' | 'skipped';

/** What an uninstall leaves in one language, as setup.Stay. */
export interface Stay {
  /** The line under Kept on the uninstall page. */
  item: string;
  /** The sentence after the uninstall. */
  note: string;
}

/** A tick of the app's own, as setup.Choice. */
export interface Choice {
  id: string;
  /** The text beside the tick by language code. */
  label: Record<string, string>;
}

/** The choices on the options page, as setup.Options. */
export interface Options {
  dir: string;
  startMenu: boolean;
  desktop: boolean;
  /** The language code the app first starts in: the one this window speaks. */
  language: string;
  /** The app's own ticks by id; a missing one is not ticked. */
  choices?: Record<string, boolean>;
}

/** The choice on the uninstall page, as setup.Removal. */
export interface Removal {
  /** Also delete what the app's window remembers; Windows only. */
  windowData: boolean;
}

/** Whether a folder may take the program, as setup.DirCheck. */
export interface DirCheck {
  ok: boolean;
  code?: string;
  detail?: string;
}

export interface Entry {
  name: string;
  path: string;
}

/** One directory as the folder chooser shows it, as setup.Folder. */
export interface Folder {
  path: string;
  /** Empty at the top of a drive. */
  parent: string;
  /** Null when the platform names none. */
  roots: string[] | null;
  entries: Entry[];
}

/** One component's part of the "progress" event, as setup.Fetched. */
export interface Fetched {
  component: string;
  state: FetchState;
  /** Bytes so far and in all; total is -1 while unknown. */
  done?: number;
  total?: number;
  error?: string;
}

/** The "progress" event: a step as it starts, a component as it moves on, or the error that ended the run. */
export type Progress = { step: string } | { error: string } | Fetched;

export interface Installer {
  info(): Promise<Info>;
  defaults(): Promise<Options>;
  checkDir(dir: string): Promise<DirCheck>;
  listDir(path: string): Promise<Folder>;
  makeDir(parent: string, name: string): Promise<string>;
  /** Returns at once; the work reports through onProgress. */
  install(options: Options): Promise<void>;
  uninstall(removal: Removal): Promise<void>;
  /** Stops fetching components and leaves the rest to the app. */
  skipComponents(): Promise<void>;
  launch(): Promise<void>;
  openURL(url: string): Promise<void>;
  minimise(): Promise<void>;
  quit(): Promise<void>;
  /** Returns the function that stops listening. */
  onProgress(listener: (p: Progress) => void): () => void;
}

interface Runtime {
  Call: { ByName(name: string, ...args: unknown[]): Promise<unknown> };
  Events: { On(name: string, callback: (event: { data: unknown }) => void): () => void };
}

// A variable, so Vite leaves the import to the browser.
const RUNTIME_URL = '/wails/runtime.js';

function isDesktop(): boolean {
  return window.location.protocol === 'wails:' || window.location.hostname === 'wails.localhost';
}

async function wailsInstaller(): Promise<Installer> {
  // Loading the runtime also tells Wails the page is ready for events.
  const runtime = (await import(/* @vite-ignore */ RUNTIME_URL)) as Runtime;
  // Wails names a bound method after the full import path of its package.
  const call = <T>(method: string, ...args: unknown[]) =>
    runtime.Call.ByName(`github.com/junkerderprovinz/pommelporter.Installer.${method}`, ...args) as Promise<T>;
  return {
    info: () => call('Info'),
    defaults: () => call('Defaults'),
    checkDir: (dir) => call('CheckDir', dir),
    listDir: (path) => call('ListDir', path),
    makeDir: (parent, name) => call('MakeDir', parent, name),
    install: (options) => call('Install', options),
    uninstall: (removal) => call('Uninstall', removal),
    skipComponents: () => call('SkipComponents'),
    launch: () => call('Launch'),
    openURL: (url) => call('OpenURL', url),
    minimise: () => call('Minimise'),
    quit: () => call('Quit'),
    onProgress: (listener) => runtime.Events.On('progress', (event) => listener(event.data as Progress)),
  };
}

export async function connect(): Promise<Installer> {
  if (import.meta.env.DEV && !isDesktop()) {
    const { mockInstaller } = await import('./mock/mock');
    return mockInstaller(new URLSearchParams(window.location.search));
  }
  return wailsInstaller();
}
