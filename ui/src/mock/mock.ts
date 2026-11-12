// A stand-in for the Installer service, so `npm run dev` shows every page in a
// plain browser. The query string picks the case:
//
//   accent=1d99f3                           the app's accent, without the #
//   mode=uninstall                          the uninstall pages
//   platform=linux                          the Linux pages, without a folder row
//   existing=1                              an older copy is installed
//   refuse=<code>                           the default folder is refused with a DirCheck code
//   fail=<step>                             the run stops with an error at that step
//   hold=<step>                             the run waits at that step, for screenshots
//   components=1                            the install fetches three components after the program
//   run=1                                   the installed program fetches one more, d, by itself
//   failComponent=<id>                      that component fails and is left for the app (a, b, c or d)
//   holdComponent=<id>                      the run waits halfway through that component
import type { Component, DirCheck, Folder, Info, Installer, Options, Progress } from '../bridge';
import { plannedSteps } from '../run';
import logo from '../assets/logo.svg?raw';

// The stand-in app of example/, so the mock and the CI installer look alike.
const app = { name: 'PommelExample', logo };

const STEP_MS = 650;

// Neutral stand-ins with sizes like those of a runtime, a codec pack and a plug-in.
const COMPONENTS: (Component & { size: number })[] = [
  { id: 'a', name: { en: 'Component A', de: 'Komponente A' }, size: 182_000_000 },
  { id: 'b', name: { en: 'Component B', de: 'Komponente B' }, size: 48_500_000 },
  { id: 'c', name: { en: 'Component C', de: 'Komponente C' }, size: 3_200_000 },
];

// One the installed program fetches itself, reporting parts instead of bytes.
const RUN_COMPONENT: Component = { id: 'd', name: { en: 'Component D', de: 'Komponente D' } };
const RUN_PARTS = 4;

/** A small disk: each folder maps to its sub-folders. */
function windowsTree(name: string, existing: boolean): Map<string, string[]> {
  return new Map([
    ['C:\\', ['Program Files', 'Program Files (x86)', 'ProgramData', 'Users', 'Windows']],
    ['C:\\Program Files', ['7-Zip', 'Common Files', ...(existing ? [name] : []), 'Mozilla Firefox', 'Windows Defender']],
    ['C:\\Users', ['Public', 'jdp']],
    ['D:\\', ['Apps', 'Games', 'Media']],
    ['D:\\Apps', []],
    ['D:\\Games', ['Steam']],
    ['D:\\Media', ['Music', 'Pictures']],
    ['E:\\', ['Photos']],
  ]);
}

export function mockInstaller(query: URLSearchParams): Installer {
  const linux = query.get('platform') === 'linux';
  const uninstall = query.get('mode') === 'uninstall';
  const hasExisting = uninstall || query.get('existing') === '1';
  const refuse = query.get('refuse');
  const fail = query.get('fail');
  const hold = query.get('hold');
  const components = !uninstall && query.get('components') === '1' ? COMPONENTS : [];
  const runComponent = !uninstall && query.get('run') === '1' ? [RUN_COMPONENT] : [];
  const listed: Component[] = [...components.map(({ id, name }) => ({ id, name })), ...runComponent];
  const failComponent = query.get('failComponent');
  const holdComponent = query.get('holdComponent');
  let skip = false;

  const defaultDir = linux ? `/home/jdp/.local/share/com.junkerderprovinz.${app.name.toLowerCase()}` : `C:\\Program Files\\${app.name}`;
  const tree = windowsTree(app.name, hasExisting);
  const listeners = new Set<(p: Progress) => void>();
  const emit = (p: Progress) => listeners.forEach((fn) => fn(p));

  const info: Info = {
    name: app.name,
    version: '2.2.0',
    publisher: 'junkerderprovinz',
    homepage: `https://github.com/junkerderprovinz/${app.name.toLowerCase()}`,
    accent: `#${query.get('accent') ?? 'FCC419'}`,
    logo: app.logo,
    mode: uninstall ? 'uninstall' : 'install',
    platform: linux ? 'linux' : 'windows',
    frameless: !linux,
    // ?choice shows an app's own tick, ?stays an app's own words for what an
    // uninstall leaves.
    stays: query.has('stays') ? { de: { item: 'Deine Einstellungen, Verknüpfungen, Farben und Schlagworte', note: 'Deine Einstellungen, Verknüpfungen, Farben und Schlagworte sind noch da.' } } : null,
    choices: query.has('choice') ? [{ id: 'folders', label: { de: 'Als Standard-Dateimanager festlegen', en: 'Make it the default file manager' } }] : null,
    existing: hasExisting ? { version: uninstall ? '2.2.0' : '2.1.0', dir: defaultDir } : null,
    components: listed,
  };

  function checkDir(dir: string): DirCheck {
    const refused = (code: string): DirCheck => ({ ok: false, code, detail: `refused: ${code}` });
    if (linux) return dir === defaultDir ? { ok: true } : refused('fixed');
    if (refuse && dir === defaultDir) return refused(refuse);
    if (!/^[A-Za-z]:\\/.test(dir)) return refused('relative');
    const clean = dir.length > 3 ? dir.replace(/\\+$/, '') : dir;
    if (clean.startsWith('E:')) return refused('notLocal');
    const children = tree.get(clean);
    if (children && children.length > 0 && clean !== defaultDir) return refused('notEmpty');
    return { ok: true };
  }

  function listDir(path: string): Folder {
    let at = path.length > 3 ? path.replace(/\\+$/, '') : path;
    while (!tree.has(at)) {
      const cut = at.lastIndexOf('\\');
      if (cut < 0) throw new Error(`no folder at or above ${path} exists`);
      at = cut <= 2 ? at.slice(0, 3) : at.slice(0, cut);
    }
    const parent = at.length > 3 ? (at.lastIndexOf('\\') <= 2 ? at.slice(0, 3) : at.slice(0, at.lastIndexOf('\\'))) : '';
    const join = (name: string) => (at.endsWith('\\') ? at + name : `${at}\\${name}`);
    return {
      path: at,
      parent,
      roots: ['C:\\', 'D:\\', 'E:\\'],
      entries: tree.get(at)!.map((name) => ({ name, path: join(name) })),
    };
  }

  const pause = (ms: number) => new Promise((r) => setTimeout(r, ms));

  async function fetchComponents() {
    for (const c of components) {
      if (skip) {
        emit({ component: c.id, state: 'skipped' });
        continue;
      }
      for (let tenth = 1; tenth <= 10; tenth++) {
        emit({ component: c.id, state: 'running', done: (c.size * tenth) / 10, total: c.size });
        if (c.id === holdComponent && tenth === 5) return false;
        if (c.id === failComponent && tenth === 3) break;
        await pause(STEP_MS / 3);
        if (skip) break;
      }
      if (c.id === failComponent) {
        emit({ component: c.id, state: 'failed', reason: 'network', error: `Get "https://example.org/component-${c.id}.zip": dial tcp: lookup example.org: no such host` });
      } else {
        emit({ component: c.id, state: skip ? 'skipped' : 'done' });
      }
    }
    for (const c of runComponent) {
      if (skip) {
        emit({ component: c.id, state: 'skipped' });
        continue;
      }
      emit({ component: c.id, state: 'running', total: -1 });
      await pause(STEP_MS * 2);
      for (let part = 1; part <= RUN_PARTS && !skip; part++) {
        emit({ component: c.id, state: 'running', status: `Part ${part} of ${RUN_PARTS}` });
        if (c.id === holdComponent && part === RUN_PARTS / 2) return false;
        if (c.id === failComponent && part === RUN_PARTS / 2) break;
        await pause(STEP_MS * 2);
        emit({ component: c.id, state: 'running', fraction: part / RUN_PARTS });
      }
      if (c.id === failComponent) {
        emit({ component: c.id, state: 'failed', reason: 'other', error: `${app.name}.exe --fetch-extras: exit status 1` });
      } else {
        emit({ component: c.id, state: skip ? 'skipped' : 'done' });
      }
    }
    return true;
  }

  async function run(mode: 'install' | 'uninstall') {
    skip = false;
    for (const step of plannedSteps(mode, info.platform, listed.length > 0)) {
      emit({ step });
      if (step === 'components' && !(await fetchComponents())) return;
      await pause(STEP_MS);
      if (step === hold) return;
      if (step === fail) {
        emit({ error: `open ${defaultDir}\\${app.name}.exe: The process cannot access the file because it is being used by another process.` });
        return;
      }
    }
    emit({ step: 'done' });
  }

  const later = <T>(value: () => T) => new Promise<T>((resolve, reject) => setTimeout(() => {
    try {
      resolve(value());
    } catch (e) {
      reject(e);
    }
  }, 120));

  return {
    info: () => later(() => info),
    defaults: () => later((): Options => ({ dir: defaultDir, startMenu: true, desktop: true, language: '' })),
    checkDir: (dir) => later(() => checkDir(dir)),
    listDir: (path) => later(() => listDir(path)),
    makeDir: (parent, name) =>
      later(() => {
        const children = tree.get(parent);
        if (!children) throw new Error(`${parent} does not exist`);
        if (children.includes(name)) throw new Error(`mkdir ${parent}\\${name}: Cannot create a file when that file already exists.`);
        const made = parent.endsWith('\\') ? parent + name : `${parent}\\${name}`;
        children.push(name);
        children.sort((a, b) => a.localeCompare(b));
        tree.set(made, []);
        return made;
      }),
    install: async () => void run('install'),
    uninstall: async () => void run('uninstall'),
    skipComponents: async () => {
      skip = true;
    },
    launch: async () => {},
    openURL: async (url) => void window.open(url, '_blank'),
    minimise: async () => {},
    quit: async () => window.location.reload(),
    onProgress: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
}
