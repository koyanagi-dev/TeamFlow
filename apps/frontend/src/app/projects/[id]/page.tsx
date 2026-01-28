import Link from "next/link";
import { apiFetch } from "@/lib/api/client";
import { normalizeApiError } from "@/lib/api/error";
import { ValidationIssues } from "@/components/ValidationIssues";
import type { ValidationIssue } from "@/lib/api/types";
import { CreateTaskForm } from "./CreateTaskForm";

// Server-side: use process.env directly (not NEXT_PUBLIC_)
const PROJECTS_BASE =
  process.env.PROJECTS_BASE ?? process.env.NEXT_PUBLIC_PROJECTS_BASE ?? "http://localhost:8080";
const TASKS_BASE =
  process.env.TASKS_BASE ?? process.env.NEXT_PUBLIC_TASKS_BASE ?? "http://localhost:8081/api";

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

function displayStatus(s: string): "todo" | "doing" | "done" {
  if (s === "done") return "done";
  if (s === "in_progress" || s === "doing") return "doing";
  return "todo";
}

function priorityOrder(p: string): number {
  switch (p) {
    case "high":
      return 3;
    case "medium":
      return 2;
    case "low":
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
    const [p, t] = await Promise.all([fetchProjects(), fetchTasksByProject(id)]);
    projects = p;
    tasks = t;
  } catch (e) {
    const n = normalizeApiError(e);
    fetchError = { message: n.message, issues: n.issues };
  }

  if (fetchError) {
    return (
      <div className="mx-auto max-w-3xl space-y-4 p-6">
        <header className="space-y-2">
          <h1 className="text-2xl font-bold">Project Detail</h1>
        </header>
        <p className="text-sm text-red-600">{fetchError.message}</p>
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
      <div className="mx-auto max-w-3xl space-y-4 p-6">
        <header className="space-y-2">
          <h1 className="text-2xl font-bold">Project Not Found</h1>
        </header>
        <p className="text-sm text-gray-700">
          ID <code className="rounded bg-gray-100 px-1 py-0.5">{id}</code>{" "}
          のプロジェクトは存在しないようです。
        </p>
        <Link href="/projects" className="text-sm text-blue-600 underline">
          ← Projects 一覧に戻る
        </Link>
      </div>
    );
  }

  const todoTasks = sortTasksByPriority(tasks.filter((t) => displayStatus(t.status) === "todo"));
  const doingTasks = sortTasksByPriority(tasks.filter((t) => displayStatus(t.status) === "doing"));
  const doneTasks = sortTasksByPriority(tasks.filter((t) => displayStatus(t.status) === "done"));

  type Col = { id: "todo" | "doing" | "done"; label: string; items: Task[] };
  const cols: Col[] = [
    { id: "todo", label: "todo", items: todoTasks },
    { id: "doing", label: "doing", items: doingTasks },
    { id: "done", label: "done", items: doneTasks },
  ];

  return (
    <div className="mx-auto max-w-5xl space-y-6 p-6">
      <header className="space-y-2">
        <Link href="/projects" className="text-sm text-blue-600 underline">
          ← Projects 一覧に戻る
        </Link>
        <h1 className="text-3xl font-bold">{project.name}</h1>
        <p className="font-mono text-xs text-gray-500">ID: {project.id}</p>
      </header>

      <section className="space-y-2 rounded-xl border bg-white p-4 shadow-sm">
        {project.description && <p className="text-sm text-gray-800">{project.description}</p>}
        <div className="space-y-1 text-xs text-gray-500">
          <div>Created: {new Date(project.createdAt).toLocaleString()}</div>
          <div>Updated: {new Date(project.updatedAt).toLocaleString()}</div>
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold">Tasks</h2>

        <CreateTaskForm projectId={id} />

        {tasks.length === 0 ? (
          <p className="text-sm text-gray-600">タスクを作成すると、ここに表示されます。</p>
        ) : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            {cols.map((col) => (
              <div key={col.id} className="rounded-xl border bg-gray-50 p-3 shadow-sm">
                <h3 className="mb-2 text-sm font-semibold text-gray-700">{col.label}</h3>
                <ul className="space-y-2">
                  {col.items.map((t) => (
                    <li
                      key={t.id}
                      className="space-y-1 rounded-lg border bg-white p-3 text-sm shadow-sm"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className="truncate font-medium">{t.title}</span>
                        <span className="shrink-0 text-[11px] text-gray-500">{t.priority}</span>
                      </div>
                      {t.description && (
                        <p className="line-clamp-2 text-gray-700">{t.description}</p>
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
