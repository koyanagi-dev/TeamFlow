import Link from 'next/link';
import { apiFetch } from '@/lib/api/client';
import { normalizeApiError } from '@/lib/api/error';
import { ValidationIssues } from '@/components/ValidationIssues';
import type { ValidationIssue } from '@/lib/api/types';
import { CreateTaskForm } from './CreateTaskForm';

// Server-side: use process.env directly (not NEXT_PUBLIC_)
const PROJECTS_BASE =
  process.env.PROJECTS_BASE ?? process.env.NEXT_PUBLIC_PROJECTS_BASE ?? 'http://localhost:8080';
const TASKS_BASE =
  process.env.TASKS_BASE ?? process.env.NEXT_PUBLIC_TASKS_BASE ?? 'http://localhost:8081/api';

type Project = {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
};

type Task = {
  id: string;
  projectId: string;
  title: string;
  description?: string;
  status: string;
  priority: string;
  createdAt: string;
  updatedAt: string;
};

async function fetchProjects(): Promise<Project[]> {
  return apiFetch<Project[]>(`${PROJECTS_BASE}/projects`);
}

async function fetchTasksByProject(projectId: string): Promise<Task[]> {
  const data = await apiFetch<Task[]>(
    `${TASKS_BASE}/tasks?projectId=${encodeURIComponent(projectId)}`
  );
  return Array.isArray(data) ? data : [];
}

function displayStatus(s: string): 'todo' | 'doing' | 'done' {
  if (s === 'done') return 'done';
  if (s === 'in_progress' || s === 'doing') return 'doing';
  return 'todo';
}

function priorityOrder(p: string): number {
  switch (p) {
    case 'high':
      return 3;
    case 'medium':
      return 2;
    case 'low':
      return 1;
    default:
      return 0;
  }
}

function sortTasksByPriority(tasks: Task[]): Task[] {
  return [...tasks].sort((a, b) => priorityOrder(b.priority) - priorityOrder(a.priority));
}

type PageProps = {
  params: Promise<{ id: string }>;
};

export default async function ProjectDetailPage({ params }: PageProps) {
  const { id } = await params;

  let projects: Project[] = [];
  let tasks: Task[] = [];
  let fetchError: { message: string; issues?: ValidationIssue[] } | null = null;

  try {
    const [p, t] = await Promise.all([
      fetchProjects(),
      fetchTasksByProject(id),
    ]);
    projects = p;
    tasks = t;
  } catch (e) {
    const n = normalizeApiError(e);
    fetchError = { message: n.message, issues: n.issues };
  }

  if (fetchError) {
    return (
      <div className="max-w-3xl mx-auto p-6 space-y-4">
        <header className="space-y-2">
          <h1 className="text-2xl font-bold">Project Detail</h1>
        </header>
        <p className="text-red-600 text-sm">{fetchError.message}</p>
        <ValidationIssues issues={fetchError.issues} />
        <Link href="/projects" className="text-sm text-blue-600 underline">
          ← Projects に戻る
        </Link>
      </div>
    );
  }

  const project = projects.find((p) => p.id === id);

  if (!project) {
    return (
      <div className="max-w-3xl mx-auto p-6 space-y-4">
        <header className="space-y-2">
          <h1 className="text-2xl font-bold">Project Not Found</h1>
        </header>
        <p className="text-sm text-gray-700">
          ID <code className="px-1 py-0.5 bg-gray-100 rounded">{id}</code>{' '}
          のプロジェクトは存在しないようです。
        </p>
        <Link href="/projects" className="text-sm text-blue-600 underline">
          ← Projects 一覧に戻る
        </Link>
      </div>
    );
  }

  const todoTasks = sortTasksByPriority(
    tasks.filter((t) => displayStatus(t.status) === 'todo')
  );
  const doingTasks = sortTasksByPriority(
    tasks.filter((t) => displayStatus(t.status) === 'doing')
  );
  const doneTasks = sortTasksByPriority(
    tasks.filter((t) => displayStatus(t.status) === 'done')
  );

  type Col = { id: 'todo' | 'doing' | 'done'; label: string; items: Task[] };
  const cols: Col[] = [
    { id: 'todo', label: 'todo', items: todoTasks },
    { id: 'doing', label: 'doing', items: doingTasks },
    { id: 'done', label: 'done', items: doneTasks },
  ];

  return (
    <div className="max-w-5xl mx-auto p-6 space-y-6">
      <header className="space-y-2">
        <Link href="/projects" className="text-sm text-blue-600 underline">
          ← Projects 一覧に戻る
        </Link>
        <h1 className="text-3xl font-bold">{project.name}</h1>
        <p className="text-xs font-mono text-gray-500">ID: {project.id}</p>
      </header>

      <section className="space-y-2 border rounded-xl p-4 bg-white shadow-sm">
        {project.description && (
          <p className="text-sm text-gray-800">{project.description}</p>
        )}
        <div className="text-xs text-gray-500 space-y-1">
          <div>Created: {new Date(project.createdAt).toLocaleString()}</div>
          <div>Updated: {new Date(project.updatedAt).toLocaleString()}</div>
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold">Tasks</h2>

        <CreateTaskForm projectId={id} />

        {tasks.length === 0 ? (
          <p className="text-sm text-gray-600">
            タスクを作成すると、ここに表示されます。
          </p>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {cols.map((col) => (
              <div
                key={col.id}
                className="border rounded-xl p-3 bg-gray-50 shadow-sm"
              >
                <h3 className="text-sm font-semibold text-gray-700 mb-2">
                  {col.label}
                </h3>
                <ul className="space-y-2">
                  {col.items.map((t) => (
                    <li
                      key={t.id}
                      className="border rounded-lg p-3 bg-white shadow-sm text-sm space-y-1"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-medium truncate">{t.title}</span>
                        <span className="text-[11px] text-gray-500 shrink-0">
                          {t.priority}
                        </span>
                      </div>
                      {t.description && (
                        <p className="text-gray-700 line-clamp-2">
                          {t.description}
                        </p>
                      )}
                      <div className="text-[11px] text-gray-500">
                        Updated: {new Date(t.updatedAt).toLocaleString()}
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
