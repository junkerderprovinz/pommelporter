// A stand-in for the Installer service, so `npm run dev` shows every page in a
// plain browser. The query string picks the case:
//
//   app=knightloader|arrowloop|trickwork   which app is being installed
//   accent=1d99f3                           the app's accent, without the #
//   mode=uninstall                          the uninstall pages
//   platform=linux                          the Linux pages, without a folder row
//   existing=1                              an older copy is installed
//   refuse=<code>                           the default folder is refused with a DirCheck code
//   fail=<step>                             the run stops with an error at that step
//   hold=<step>                             the run waits at that step, for screenshots
import type { DirCheck, Folder, Info, Installer, Options, Progress } from '../bridge';
import { plannedSteps } from '../run';
import knightloader from './knightloader.svg?raw';
import arrowloop from './arrowloop.svg?raw';
import trickwork from './trickwork.svg?raw';

const APPS: Record<string, { name: string; logo: string }> = {
  knightloader: { name: 'KnightLoader', logo: knightloader },
  arrowloop: { name: 'ArrowLoop', logo: arrowloop },
  trickwork: { name: 'TrickWork', logo: trickwork },
};

const STEP_MS = 650;

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
  const app = APPS[query.get('app') ?? ''] ?? APPS.knightloader!;
  const linux = query.get('platform') === 'linux';
  const uninstall = query.get('mode') === 'uninstall';
  const hasExisting = uninstall || query.get('existing') === '1';
  const refuse = query.get('refuse');
  const fail = query.get('fail');
  const hold = query.get('hold');

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
    existing: hasExisting ? { version: uninstall ? '2.2.0' : '2.1.0', dir: defaultDir } : null,
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

  async function run(mode: 'install' | 'uninstall') {
    for (const step of plannedSteps(mode, info.platform)) {
      emit({ step });
      await new Promise((r) => setTimeout(r, STEP_MS));
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
