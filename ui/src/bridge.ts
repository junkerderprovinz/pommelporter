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
}

/** The choices on the options page, as setup.Options. */
export interface Options {
  dir: string;
  startMenu: boolean;
  desktop: boolean;
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

/** The "progress" event: a step as it starts, or the error that ended the run. */
export type Progress = { step: string } | { error: string };

export interface Installer {
  info(): Promise<Info>;
  defaults(): Promise<Options>;
  checkDir(dir: string): Promise<DirCheck>;
  listDir(path: string): Promise<Folder>;
  makeDir(parent: string, name: string): Promise<string>;
  /** Returns at once; the work reports through onProgress. */
  install(options: Options): Promise<void>;
  uninstall(): Promise<void>;
  launch(): Promise<void>;
  openURL(url: string): Promise<void>;
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
  const call = <T>(method: string, ...args: unknown[]) =>
    runtime.Call.ByName(`pommelporter.Installer.${method}`, ...args) as Promise<T>;
  return {
    info: () => call('Info'),
    defaults: () => call('Defaults'),
    checkDir: (dir) => call('CheckDir', dir),
    listDir: (path) => call('ListDir', path),
    makeDir: (parent, name) => call('MakeDir', parent, name),
    install: (options) => call('Install', options),
    uninstall: () => call('Uninstall'),
    launch: () => call('Launch'),
    openURL: (url) => call('OpenURL', url),
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
