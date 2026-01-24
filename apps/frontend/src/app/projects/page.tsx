import Link from 'next/link';

type Project = {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
};

// ここから下は前回のままでOK（fetchProjects などは変更なし）

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

export default async function ProjectsPage() {
  let projects: Project[] = [];

  try {
    projects = await fetchProjects();
  } catch {
    return (
      <div className="max-w-3xl mx-auto p-6">
        <h1 className="text-2xl font-bold mb-4">Projects</h1>
        <p className="text-red-600 text-sm">
          プロジェクト一覧の取得に失敗しました。
          バックエンド（Go projects サービス）が起動しているか確認してください。
        </p>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      <header className="space-y-2">
        <h1 className="text-3xl font-bold">Projects</h1>
        <p className="text-sm text-gray-600">
          TeamFlow 上のプロジェクト一覧です（現在は読み取り専用 / ローカル環境）。
        </p>
      </header>

      {projects.length === 0 ? (
        <p className="text-sm text-gray-600">
          まだプロジェクトがありません。<br />
          開発用ページ <code className="px-1 py-0.5 bg-gray-100 rounded">/dev/projects</code>{' '}
          からプロジェクトを作成してみてください。
        </p>
      ) : (
        <ul className="space-y-3">
          {projects.map((p) => (
            <li key={p.id}>
              <Link
                href={`/projects/${p.id}`}
                className="block border rounded-xl p-4 bg-white shadow-sm hover:shadow-md transition-shadow"
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs font-mono text-gray-500">
                    ID: {p.id}
                  </span>
                  <span className="text-[11px] text-gray-500">
                    Updated: {new Date(p.updatedAt).toLocaleString()}
                  </span>
                </div>
                <div className="text-lg font-semibold mt-1">{p.name}</div>
                {p.description && (
                  <p className="text-sm text-gray-700 mt-1">
                    {p.description}
                  </p>
                )}
              </Link>
            </li>
          ))}
        </ul>
      )}

      <footer className="text-xs text-gray-500 border-t pt-3">
        開発用フォーム:{' '}
        <code className="px-1 py-0.5 bg-gray-100 rounded">
          /dev/projects
        </code>
      </footer>
    </div>
  );
}
