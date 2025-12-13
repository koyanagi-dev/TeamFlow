'use client';

import { useState } from 'react';

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

function generateTaskId() {
  // 開発用なのでざっくりユニークなら OK
  return 'task-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 6);
}

export default function DevTasksPage() {
  const [id, setId] = useState<string>(generateTaskId());
  const [projectId, setProjectId] = useState<string>('proj-1');
  const [title, setTitle] = useState<string>('タスクのタイトル');
  const [description, setDescription] = useState<string>('タスクの説明');
  const [status, setStatus] = useState<string>('todo');
  const [priority, setPriority] = useState<string>('medium');

  const [loading, setLoading] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [lastCreated, setLastCreated] = useState<Task | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setCreateError(null);
    setLastCreated(null);

    try {
      const res = await fetch('/api/dev/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id,
          projectId,
          title,
          description,
          status,
          priority,
        }),
      });

      const text = await res.text();
      if (!res.ok) throw new Error(text);

      const data = JSON.parse(text) as Task;
      setLastCreated(data);

      // 送信が成功したら次のタスク用に ID を自動で振り直す
      setId(generateTaskId());
    } catch (err: any) {
      setCreateError(err.message ?? String(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-xl mx-auto p-4 space-y-8">
      <h1 className="text-2xl font-bold">Dev: Tasks</h1>

      <form onSubmit={handleSubmit} className="space-y-3 border rounded-lg p-4">
        <div>
          <label className="block text-sm font-medium mb-1">Task ID</label>
          <input
            className="w-full border rounded px-2 py-1 text-sm"
            value={id}
            onChange={(e) => setId(e.target.value)}
          />
          <p className="text-[11px] text-gray-500 mt-1">
            開発用フォームのため、送信成功時に自動で新しい ID が採番されます。
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Project ID</label>
          <input
            className="w-full border rounded px-2 py-1 text-sm"
            value={projectId}
            onChange={(e) => setProjectId(e.target.value)}
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Title</label>
          <input
            className="w-full border rounded px-2 py-1 text-sm"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Description</label>
          <textarea
            className="w-full border rounded px-2 py-1 text-sm"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>

        <div className="flex gap-3">
          <div className="flex-1">
            <label className="block text-sm font-medium mb-1">Status</label>
            <input
              className="w-full border rounded px-2 py-1 text-sm"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
            />
          </div>
          <div className="flex-1">
            <label className="block text-sm font-medium mb-1">Priority</label>
            <input
              className="w-full border rounded px-2 py-1 text-sm"
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="px-4 py-2 border rounded text-sm disabled:opacity-60"
        >
          {loading ? 'Sending...' : 'Create Task'}
        </button>
      </form>

      {createError && (
        <div className="text-sm text-red-600 whitespace-pre-wrap">
          Error: {createError}
        </div>
      )}

      {lastCreated && (
        <div className="text-sm border rounded-lg p-3 bg-gray-50 space-y-1">
          <div><span className="font-medium">ID:</span> {lastCreated.id}</div>
          <div><span className="font-medium">ProjectID:</span> {lastCreated.projectId}</div>
          <div><span className="font-medium">Title:</span> {lastCreated.title}</div>
          <div><span className="font-medium">Status:</span> {lastCreated.status}</div>
          <div><span className="font-medium">Priority:</span> {lastCreated.priority}</div>
        </div>
      )}
    </div>
  );
}
