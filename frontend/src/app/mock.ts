import { AppInfo, ClaudeStatus, NotificationStatus, Profile, Settings, Subtask, SyncStatus, Task, TaskImage, TaskInput, UpdateInfo } from './models';

/**
 * In-memory fallback used when the Angular app runs in a plain browser
 * (`npm start`) without the Go/Wails bridge. It keeps the UI fully usable for
 * design work and previews; the desktop app always uses the Go backend.
 */

const DAY = 86_400_000;
let counter = 0;
const id = () => `mock-${Date.now().toString(36)}-${counter++}`;
const wait = (ms = 140) => new Promise<void>((resolve) => setTimeout(resolve, ms));
const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T;

function at(daysFromNow: number, hours: number, minutes = 0): number {
  const date = new Date();
  date.setDate(date.getDate() + daysFromNow);
  date.setHours(hours, minutes, 0, 0);
  return date.getTime();
}

function placeholder(label: string, from: string, to: string): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="900" height="640" viewBox="0 0 900 640">
    <defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="${from}"/><stop offset="1" stop-color="${to}"/>
    </linearGradient></defs>
    <rect width="900" height="640" fill="url(#g)"/>
    <circle cx="720" cy="120" r="150" fill="#ffffff" opacity=".22"/>
    <circle cx="120" cy="560" r="190" fill="#ffffff" opacity=".16"/>
    <rect x="80" y="90" width="430" height="26" rx="13" fill="#ffffff" opacity=".55"/>
    <rect x="80" y="140" width="280" height="16" rx="8" fill="#ffffff" opacity=".4"/>
    <rect x="80" y="420" width="360" height="120" rx="22" fill="#ffffff" opacity=".35"/>
    <text x="110" y="495" font-family="Inter, sans-serif" font-size="30" font-weight="600" fill="#3c3550">${label}</text>
  </svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

function sub(title: string, done = false, position = 0): Subtask {
  return { id: id(), title, done, position };
}

function image(name: string, from: string, to: string): TaskImage {
  return { id: id(), name, dataUrl: placeholder(name, from, to), createdAt: Date.now() - 2 * DAY };
}

const profileColors = ['#7b6cf0', '#4aa985', '#559fe0', '#ea8a62', '#dd7ba2', '#d3a640'];
const persoProfile = 'mock-profile-perso';
const travailProfile = 'mock-profile-travail';
let profiles: Profile[] = [
  { id: persoProfile, name: 'Perso', color: profileColors[0], position: 0, createdAt: Date.now() - 30 * DAY },
  { id: travailProfile, name: 'Travail', color: profileColors[1], position: 1, createdAt: Date.now() - 30 * DAY },
];

function seed(): Task[] {
  const now = Date.now();
  return [
    {
      id: id(),
      profileId: persoProfile,
      title: 'Préparer la démo de vendredi',
      description: 'Relire le scénario, vérifier les chiffres et tenir en 10 minutes. Garder trois slides maximum.',
      done: false,
      position: 0,
      createdAt: now - 2 * DAY,
      dueAt: at(0, 23, 30),
      completedAt: null,
      startedAt: now - 3_600_000,
      timeSpent: 0,
      subtasks: [
        sub('Écrire le fil rouge', true, 0),
        sub('Vérifier les chiffres clés', true, 1),
        sub('Préparer les trois slides', false, 2),
        sub('Répéter une fois à voix haute', false, 3),
      ],
      images: [],
    },
    {
      id: id(),
      profileId: travailProfile,
      title: 'Relire les maquettes du nouvel onboarding',
      description: 'Se concentrer sur le premier écran et le moment où on demande le nom de l’équipe.',
      done: false,
      position: 1,
      createdAt: now - 4 * DAY,
      dueAt: at(1, 11, 30),
      completedAt: null,
      startedAt: null,
      timeSpent: 0,
      subtasks: [sub('Annoter les écrans', true, 0), sub('Regrouper les retours', false, 1), sub('Envoyer à Camille', false, 2)],
      images: [image('Onboarding · écran 1', '#e7e1ff', '#f7d9e8'), image('Onboarding · écran 2', '#dff3ec', '#e4ecff')],
    },
    {
      id: id(),
      profileId: travailProfile,
      title: 'Écrire le compte-rendu de la réunion produit',
      description: 'Décisions, questions ouvertes et prochaines étapes. Cinq lignes suffisent.',
      done: false,
      position: 0,
      createdAt: now - 6 * DAY,
      dueAt: at(-1, 18, 0),
      completedAt: null,
      startedAt: null,
      timeSpent: 0,
      subtasks: [],
      images: [],
    },
    {
      id: id(),
      profileId: persoProfile,
      title: 'Choisir un cadeau pour Léa',
      description: 'Quelque chose de simple, utile, et qui ne prend pas de place.',
      done: false,
      position: 1,
      createdAt: now - 5 * DAY,
      dueAt: null,
      completedAt: null,
      startedAt: null,
      timeSpent: 0,
      subtasks: [sub('Lister deux ou trois idées', false, 0)],
      images: [],
    },
    {
      id: id(),
      profileId: persoProfile,
      title: 'Répondre aux e-mails en attente',
      description: '',
      done: true,
      position: 2,
      createdAt: now - 1 * DAY,
      dueAt: null,
      completedAt: at(0, 10, 12),
      startedAt: null,
      timeSpent: 0,
      subtasks: [],
      images: [],
    },
    {
      id: id(),
      profileId: travailProfile,
      title: 'Mettre à jour le board de l’équipe',
      description: '',
      done: true,
      position: 2,
      createdAt: now - 2 * DAY,
      dueAt: null,
      completedAt: at(0, 9, 5),
      startedAt: null,
      timeSpent: 0,
      subtasks: [],
      images: [],
    },
    {
      id: id(),
      profileId: travailProfile,
      title: 'Envoyer les factures de septembre',
      description: '',
      done: true,
      position: 3,
      createdAt: now - 4 * DAY,
      dueAt: null,
      completedAt: at(-1, 17, 40),
      startedAt: null,
      timeSpent: 0,
      subtasks: [],
      images: [],
    },
    {
      id: id(),
      profileId: persoProfile,
      title: 'Nettoyer les notes de la semaine',
      description: '',
      done: true,
      position: 3,
      createdAt: now - 7 * DAY,
      dueAt: null,
      completedAt: at(-3, 18, 20),
      startedAt: null,
      timeSpent: 0,
      subtasks: [],
      images: [],
    },
  ];
}

let tasks = seed();
let settings: Settings = loadSettings();
let syncStatus: SyncStatus = {
  configured: false,
  remote: '',
  branch: 'main',
  enabled: false,
  syncing: false,
  lastSyncAt: 0,
  lastError: '',
  generation: 0,
  gitAvailable: true,
};

function loadSettings(): Settings {
  const fallback: Settings = { accent: 'lavande', coachEnabled: true, coachOnStart: false, timeTracking: true, showMomentum: true, notifications: true, autoUpdate: true, claudePath: '' };
  try {
    const raw = localStorage.getItem('todo.settings');
    return raw ? { ...fallback, ...(JSON.parse(raw) as Partial<Settings>) } : fallback;
  } catch {
    return fallback;
  }
}

function bank(task: Task): void {
  if (task.startedAt !== null) {
    task.timeSpent += Math.max(0, Date.now() - task.startedAt);
    task.startedAt = null;
  }
}

function findTask(taskId: string): Task {
  const task = tasks.find((candidate) => candidate.id === taskId);
  if (!task) {
    throw new Error('Tâche introuvable');
  }
  return task;
}

function findSubtask(subtaskId: string): { task: Task; subtask: Subtask } {
  for (const task of tasks) {
    const subtask = task.subtasks.find((candidate) => candidate.id === subtaskId);
    if (subtask) {
      return { task, subtask };
    }
  }
  throw new Error('Sous-tâche introuvable');
}

function nextTopPosition(profileId: string): number {
  const positions = tasks.filter((task) => task.profileId === profileId).map((task) => task.position);
  return (positions.length > 0 ? Math.min(...positions) : 0) - 1;
}

function createTask(input: TaskInput): Task {
  const profileId = profiles.some((profile) => profile.id === input.profileId)
    ? input.profileId
    : (profiles[0]?.id ?? '');
  const task: Task = {
    id: id(),
    profileId,
    title: input.title.trim(),
    description: input.description,
    done: false,
    position: nextTopPosition(profileId),
    createdAt: Date.now(),
    dueAt: input.dueAt,
    completedAt: null,
    startedAt: null,
    timeSpent: 0,
    subtasks: [],
    images: [],
  };
  tasks = [task, ...tasks];
  return task;
}

export async function mockCall<T>(method: string, args: unknown[]): Promise<T> {
  await wait(method === 'BreakDownTask' ? 1200 : method === 'CoachNudge' ? 1400 : 90);

  switch (method) {
    case 'ListTasks':
      return clone(tasks) as T;
    case 'CreateTask':
      return clone(createTask(args[0] as TaskInput)) as T;
    case 'UpdateTask': {
      const task = findTask(args[0] as string);
      const input = args[1] as TaskInput;
      if (input.profileId && input.profileId !== task.profileId) {
        task.profileId = input.profileId;
        task.position = nextTopPosition(input.profileId);
      }
      task.title = input.title.trim();
      task.description = input.description;
      task.dueAt = input.dueAt;
      return clone(task) as T;
    }
    case 'SetTaskDone': {
      const task = findTask(args[0] as string);
      task.done = args[1] as boolean;
      if (task.done) {
        task.completedAt = Date.now();
      } else {
        task.completedAt = null;
        task.position = nextTopPosition(task.profileId);
      }
      bank(task);
      return clone(task) as T;
    }
    case 'SetTaskStarted': {
      const task = findTask(args[0] as string);
      if (args[1]) {
        task.startedAt ??= task.done ? null : Date.now();
      } else {
        bank(task);
      }
      return clone(task) as T;
    }
    case 'DeleteTask':
      tasks = tasks.filter((task) => task.id !== args[0]);
      return undefined as T;
    case 'ReorderTasks': {
      const ids = args[0] as string[];
      ids.forEach((taskId, index) => {
        const task = tasks.find((candidate) => candidate.id === taskId);
        if (task) {
          task.position = index;
        }
      });
      return undefined as T;
    }
    case 'AddSubtask': {
      const task = findTask(args[0] as string);
      task.subtasks.push({ id: id(), title: args[1] as string, done: false, position: task.subtasks.length });
      return clone(task) as T;
    }
    case 'AddSubtasks': {
      const task = findTask(args[0] as string);
      for (const title of args[1] as string[]) {
        task.subtasks.push({ id: id(), title, done: false, position: task.subtasks.length });
      }
      return clone(task) as T;
    }
    case 'UpdateSubtask': {
      const { task, subtask } = findSubtask(args[0] as string);
      subtask.title = args[1] as string;
      return clone(task) as T;
    }
    case 'SetSubtaskDone': {
      const { task, subtask } = findSubtask(args[0] as string);
      subtask.done = args[1] as boolean;
      return clone(task) as T;
    }
    case 'DeleteSubtask': {
      let owner: Task | undefined;
      for (const task of tasks) {
        if (task.subtasks.some((candidate) => candidate.id === args[0])) {
          owner = task;
        }
      }
      if (!owner) {
        throw new Error('Sous-tâche introuvable');
      }
      owner.subtasks = owner.subtasks.filter((candidate) => candidate.id !== args[0]);
      return clone(owner) as T;
    }
    case 'ReorderSubtasks': {
      const task = findTask(args[0] as string);
      const ids = args[1] as string[];
      ids.forEach((subtaskId, index) => {
        const subtask = task.subtasks.find((candidate) => candidate.id === subtaskId);
        if (subtask) {
          subtask.position = index;
        }
      });
      return clone(task) as T;
    }
    case 'AddImage': {
      const task = findTask(args[0] as string);
      const palettes = [
        ['#ece8ff', '#fbe9f0'],
        ['#e3f4ec', '#e6f1fb'],
        ['#fceee7', '#f9f1da'],
      ];
      const colors = palettes[task.images.length % palettes.length];
      task.images.push(image(`Image ${task.images.length + 1}`, colors[0], colors[1]));
      return clone(task) as T;
    }
    case 'DeleteImage': {
      let owner: Task | undefined;
      for (const task of tasks) {
        if (task.images.some((candidate) => candidate.id === args[0])) {
          owner = task;
        }
      }
      if (!owner) {
        throw new Error('Image introuvable');
      }
      owner.images = owner.images.filter((candidate) => candidate.id !== args[0]);
      return clone(owner) as T;
    }
    case 'ListProfiles':
      return clone(profiles) as T;
    case 'CreateProfile': {
      const name = String(args[0] ?? '').trim();
      if (!name) {
        throw new Error('Le nom du profil est obligatoire');
      }
      const profile: Profile = {
        id: id(),
        name: name.slice(0, 40),
        color: profileColors[profiles.length % profileColors.length],
        position: profiles.length,
        createdAt: Date.now(),
      };
      profiles = [...profiles, profile];
      return clone(profile) as T;
    }
    case 'UpdateProfile': {
      const profile = profiles.find((candidate) => candidate.id === args[0]);
      if (!profile) {
        throw new Error('Profil introuvable');
      }
      const name = String(args[1] ?? '').trim();
      if (!name) {
        throw new Error('Le nom du profil est obligatoire');
      }
      profile.name = name.slice(0, 40);
      const color = String(args[2] ?? '').trim();
      if (color) {
        profile.color = color;
      }
      return clone(profile) as T;
    }
    case 'DeleteProfile': {
      if (profiles.length <= 1) {
        throw new Error('Il faut conserver au moins un profil');
      }
      const replacement = profiles.find((candidate) => candidate.id !== args[0]);
      tasks = tasks.map((task) => (task.profileId === args[0] ? { ...task, profileId: replacement?.id ?? task.profileId } : task));
      profiles = profiles.filter((candidate) => candidate.id !== args[0]);
      return undefined as T;
    }
    case 'NotificationStatus':
    case 'RequestNotifications': {
      const status: NotificationStatus = {
        available: true,
        authorized: true,
        error: '',
      };
      return status as T;
    }
    case 'TestNotification':
      return undefined as T;
    case 'GetSettings':
      return clone(settings) as T;
    case 'SaveSettings': {
      settings = clone(args[0] as Settings);
      localStorage.setItem('todo.settings', JSON.stringify(settings));
      return undefined as T;
    }
    case 'ClaudeStatus': {
      const status: ClaudeStatus = {
        available: true,
        path: '/Users/you/.local/bin/claude',
        version: '2.1.312 (aperçu navigateur)',
        error: '',
      };
      return status as T;
    }
    case 'BreakDownTask': {
      const task = findTask(args[0] as string);
      const existing = new Set(task.subtasks.map((item) => item.title.toLowerCase()));
      const suggestions = [
        'Définir le résultat attendu',
        'Lister les étapes clés',
        'Bloquer 30 minutes dans l’agenda',
        'Vérifier et ajuster',
      ].filter((item) => !existing.has(item.toLowerCase()));
      return suggestions as T;
    }
    case 'CoachNudge': {
      const open = tasks.filter((task) => !task.done);
      const overdue = open.filter((task) => task.dueAt !== null && task.dueAt < Date.now());
      if (overdue.length > 0) {
        return `Une seule chose à la fois : ${overdue[0].title.toLowerCase()} attend depuis un moment. Ouvre-la et passe juste cinq minutes dessus, tu verras que le reste suivra.` as T;
      }
      return `Ta liste avance bien, et c'est déjà beaucoup. Choisis la tâche la plus courte et fais-la maintenant, sans chercher à tout finir aujourd'hui.` as T;
    }
    case 'SyncStatus':
      return clone(syncStatus) as T;
    case 'SetSyncConfig': {
      const remote = String(args[0] ?? '').trim();
      const enabled = Boolean(args[1]);
      syncStatus = {
        ...syncStatus,
        configured: remote !== '',
        remote,
        enabled: remote !== '' && enabled,
        lastError: '',
        generation: syncStatus.generation + 1,
      };
      return undefined as T;
    }
    case 'SyncNow':
      if (!syncStatus.configured) {
        throw new Error('Aucun dépôt git configuré');
      }
      syncStatus = { ...syncStatus, syncing: true };
      await wait(700);
      syncStatus = {
        ...syncStatus,
        syncing: false,
        lastSyncAt: Date.now(),
        lastError: '',
        generation: syncStatus.generation + 1,
      };
      return clone(syncStatus) as T;
    case 'OpenSyncFolder':
      return undefined as T;
    case 'OpenDataFolder':
      return undefined as T;
    case 'UpdateStatus':
    case 'CheckForUpdates':
    case 'DownloadUpdate': {
      const info: UpdateInfo = {
        currentVersion: '1.2.2 · aperçu navigateur',
        latestVersion: '',
        available: false,
        downloading: false,
        ready: false,
        notes: '',
        releaseUrl: 'https://github.com/clementherve/todo/releases',
        publishedAt: 0,
        error: '',
      };
      return info as T;
    }
    case 'InstallUpdate':
    case 'OpenUpdatePage':
      return undefined as T;
    case 'AppInfo': {
      const info: AppInfo = {
        version: '1.2.2 · aperçu navigateur',
        dataDir: '~/Library/Application Support/todo',
        database: 'todo.db',
        databasePath: '~/Library/Application Support/todo/todo.db',
      };
      return info as T;
    }
    default:
      throw new Error(`Méthode inconnue : ${method}`);
  }
}
