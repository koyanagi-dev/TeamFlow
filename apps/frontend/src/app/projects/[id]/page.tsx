import Link from 'next/link';

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
  const res = await fetch('http://localhost:8080/projects', {
    cache: 'no-store',
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to load projects: ${res.status} ${text}`);
  }
  return res.json();
}

async function fetchTasksByProject(projectId: string): Promise<Task[]> {
  const res = await fetch(
    `http://localhost:8081/tasks?projectId=${encodeURIComponent(projectId)}`,
    { cache: 'no-store' },
  );
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to load tasks: ${res.status} ${text}`);
  }
  return res.json();
}

type PageProps = {
  params: Promise<{ id: string }>;
};

export default async function ProjectDetailPage({ params }: PageProps) {
  const { id } = await params;

  let projects: Project[] = [];
  let tasks: Task[] = [];

  try {
    const [p, t] = await Promise.all([
      fetchProjects(),
      fetchTasksByProject(id),
    ]);
    projects = p;
    tasks = t;
  } catch (e) {
    return (
      <div className="max-w-3xl mx-auto p-6 space-y-4">
        <header className="space-y-2">
          <h1 className="text-2xl font-bold">Project Detail</h1>
        </header>
        <p className="text-red-600 text-sm">
          プロジェクトまたはタスク情報の取得に失敗しました。
          バックエンド（Go services）が起動しているか確認してください。
        </p>
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

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      <header className="space-y-2">
        <Link href="/projects" className="text-sm text-blue-600 underline">
          ← Projects 一覧に戻る
        </Link>
        <h1 className="text-3xl font-bold">{project.name}</h1>
        <p className="text-xs font-mono text-gray-500">ID: {project.id}</p>
      </header>

      {/* プロジェクト概要 */}
      <section className="space-y-2 border rounded-xl p-4 bg-white shadow-sm">
        {project.description && (
          <p className="text-sm text-gray-800">{project.description}</p>
        )}
        <div className="text-xs text-gray-500 space-y-1">
          <div>
            Created:{' '}
            {new Date(project.createdAt).toLocaleString()}
          </div>
          <div>
            Updated:{' '}
            {new Date(project.updatedAt).toLocaleString()}
          </div>
        </div>
      </section>

      {/* タスク一覧 */}
      <section className="space-y-3">
        <h2 className="text-xl font-semibold">Tasks</h2>

        {tasks.length === 0 ? (
          <p className="text-sm text-gray-600">
            このプロジェクトに紐づくタスクはまだありません。<br />
            開発用ページ{' '}
            <code className="px-1 py-0.5 bg-gray-100 rounded">
              /dev/tasks
            </code>{' '}
            から projectId = {project.id} のタスクを作成してみてください。
          </p>
        ) : (
          <ul className="space-y-2">
            {tasks.map((t) => (
              <li
                key={t.id}
                className="border rounded-lg p-3 bg-white shadow-sm text-sm space-y-1"
              >
                <div className="flex items-center justify-between">
                  <span className="font-medium">{t.title}</span>
                  <span className="text-[11px] text-gray-500">
                    {t.status} / {t.priority}
                  </span>
                </div>
                {t.description && (
                  <p className="text-gray-700">{t.description}</p>
                )}
                <div className="text-[11px] text-gray-500">
                  Updated: {new Date(t.updatedAt).toLocaleString()}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
