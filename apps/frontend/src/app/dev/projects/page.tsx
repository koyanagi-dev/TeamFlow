"use client";

import { useState } from "react";

type Project = {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
};

export default function DevProjectsPage() {
  // --- Create Project ---
  const [id, setId] = useState("proj-1");
  const [name, setName] = useState("TeamFlow 開発");
  const [description, setDescription] = useState("TeamFlow の開発プロジェクト");
  const [loading, setLoading] = useState(false);
  const [createResult, setCreateResult] = useState<Project | null>(null);
  const [createError, setCreateError] = useState<string | null>(null);

  // --- List Projects ---
  const [listLoading, setListLoading] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);
  const [listError, setListError] = useState<string | null>(null);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setCreateError(null);
    setCreateResult(null);

    try {
      const res = await fetch("/api/dev/projects", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id, name, description }),
      });

      const text = await res.text();
      if (!res.ok) throw new Error(text);

      const data = JSON.parse(text) as Project;
      setCreateResult(data);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      setCreateError(message);
    } finally {
      setLoading(false);
    }
  };

  const handleLoadProjects = async () => {
    setListLoading(true);
    setListError(null);

    try {
      const res = await fetch("/api/dev/projects/list");
      const text = await res.text();
      if (!res.ok) throw new Error(text);

      const data = JSON.parse(text) as Project[];
      setProjects(data);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      setListError(message);
    } finally {
      setListLoading(false);
    }
  };

  return (
    <div className="mx-auto max-w-xl space-y-10 p-4">
      <h1 className="text-2xl font-bold">Dev: Create Project</h1>

      {/* --- Create Form --- */}
      <form onSubmit={handleCreate} className="space-y-3 rounded-lg border p-4">
        <div>
          <label className="mb-1 block text-sm font-medium">Project ID</label>
          <input
            className="w-full rounded border px-2 py-1 text-sm"
            value={id}
            onChange={(e) => setId(e.target.value)}
          />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium">Name</label>
          <input
            className="w-full rounded border px-2 py-1 text-sm"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium">Description</label>
          <textarea
            className="w-full rounded border px-2 py-1 text-sm"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="rounded border px-4 py-2 text-sm disabled:opacity-60"
        >
          {loading ? "Sending..." : "Create Project"}
        </button>
      </form>

      {createError && (
        <div className="whitespace-pre-wrap text-sm text-red-600">Error: {createError}</div>
      )}

      {createResult && (
        <div className="space-y-1 rounded-lg border bg-gray-50 p-3 text-sm">
          <div>
            <span className="font-medium">ID:</span> {createResult.id}
          </div>
          <div>
            <span className="font-medium">Name:</span> {createResult.name}
          </div>
          <div>
            <span className="font-medium">Description:</span> {createResult.description}
          </div>
          <div>
            <span className="font-medium">CreatedAt:</span> {createResult.createdAt}
          </div>
          <div>
            <span className="font-medium">UpdatedAt:</span> {createResult.updatedAt}
          </div>
        </div>
      )}

      {/* --- List Projects --- */}
      <div className="space-y-4">
        <h2 className="text-xl font-bold">Project List</h2>

        <button
          onClick={handleLoadProjects}
          disabled={listLoading}
          className="rounded border px-4 py-2 text-sm disabled:opacity-60"
        >
          {listLoading ? "Loading..." : "Load Projects"}
        </button>

        {listError && (
          <div className="whitespace-pre-wrap text-sm text-red-600">Error: {listError}</div>
        )}

        {projects.length > 0 && (
          <ul className="space-y-2">
            {projects.map((p) => (
              <li key={p.id} className="rounded-lg border bg-gray-50 p-3">
                <div>
                  <span className="font-medium">ID:</span> {p.id}
                </div>
                <div>
                  <span className="font-medium">Name:</span> {p.name}
                </div>
                <div>
                  <span className="font-medium">Desc:</span> {p.description}
                </div>
                <div>
                  <span className="font-medium">Updated:</span> {p.updatedAt}
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
