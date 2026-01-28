import Link from "next/link";

type Project = {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
};

// ここから下は前回のままでOK（fetchProjects などは変更なし）

async function fetchProjects(): Promise<Project[]> {
  const res = await fetch("http://localhost:8080/projects", {
    cache: "no-store",
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
      <div className="mx-auto max-w-3xl p-6">
        <h1 className="mb-4 text-2xl font-bold">Projects</h1>
        <p className="text-sm text-red-600">
          プロジェクト一覧の取得に失敗しました。 バックエンド（Go projects
          サービス）が起動しているか確認してください。
        </p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      <header className="space-y-2">
        <h1 className="text-3xl font-bold">Projects</h1>
        <p className="text-sm text-gray-600">
          TeamFlow 上のプロジェクト一覧です（現在は読み取り専用 / ローカル環境）。
        </p>
      </header>

      {projects.length === 0 ? (
        <p className="text-sm text-gray-600">
          まだプロジェクトがありません。
          <br />
          開発用ページ <code className="rounded bg-gray-100 px-1 py-0.5">/dev/projects</code>{" "}
          からプロジェクトを作成してみてください。
        </p>
      ) : (
        <ul className="space-y-3">
          {projects.map((p) => (
            <li key={p.id}>
              <Link
                href={`/projects/${p.id}`}
                className="block rounded-xl border bg-white p-4 shadow-sm transition-shadow hover:shadow-md"
              >
                <div className="flex items-center justify-between">
                  <span className="font-mono text-xs text-gray-500">ID: {p.id}</span>
                  <span className="text-[11px] text-gray-500">
                    Updated: {new Date(p.updatedAt).toLocaleString()}
                  </span>
                </div>
                <div className="mt-1 text-lg font-semibold">{p.name}</div>
                {p.description && <p className="mt-1 text-sm text-gray-700">{p.description}</p>}
              </Link>
            </li>
          ))}
        </ul>
      )}

      <footer className="border-t pt-3 text-xs text-gray-500">
        開発用フォーム: <code className="rounded bg-gray-100 px-1 py-0.5">/dev/projects</code>
      </footer>
    </div>
  );
}
