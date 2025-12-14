'use client';

import { useEffect, useMemo, useState } from 'react';

type Task = {
  id: string;
  projectId: string;
  title: string;
  description?: string;
  status: string;
  priority: string;
  dueDate?: string | null;
  createdAt: string;
  updatedAt: string;
};

function generateTaskId() {
  return 'task-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 6);
}

function TaskRow({ task, onUpdateSuccess }: { task: Task; onUpdateSuccess: () => void }) {
  // Map "in_progress" from API response to "doing" for UI
  const normalizeStatus = (s: string) => s === 'in_progress' ? 'doing' : s;
  const denormalizeStatus = (s: string) => s === 'doing' ? 'in_progress' : s;

  const [titleInput, setTitleInput] = useState<string>(task.title);
  const [statusInput, setStatusInput] = useState<string>(normalizeStatus(task.status));
  const [priorityInput, setPriorityInput] = useState<string>(task.priority);
  const [updating, setUpdating] = useState(false);
  const [updateError, setUpdateError] = useState<string | null>(null);

  // task が変わったら inputs も更新（外部から更新された場合）
  useEffect(() => {
    setTitleInput(task.title);
    setStatusInput(normalizeStatus(task.status));
    setPriorityInput(task.priority);
  }, [task.title, task.status, task.priority]);

  const handleUpdate = async () => {
    setUpdating(true);
    setUpdateError(null);

    try {
      const body: { title?: string; status?: string; priority?: string } = {};
      
      // 変更されたフィールドだけを追加
      if (titleInput !== task.title) {
        body.title = titleInput;
      }
      if (statusInput !== normalizeStatus(task.status)) {
        // API に送る時は "doing" をそのまま送る（ハンドラ側で "in_progress" にマッピングされる）
        body.status = statusInput;
      }
      if (priorityInput !== task.priority) {
        body.priority = priorityInput;
      }

      // 変更がない場合は何もしない
      if (Object.keys(body).length === 0) {
        setUpdating(false);
        return;
      }

      const res = await fetch(`/api/dev/tasks?id=${encodeURIComponent(task.id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      const text = await res.text();
      if (!res.ok) throw new Error(text);

      // 成功したら一覧を再取得
      onUpdateSuccess();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setUpdateError(msg);
      alert(`Failed to update: ${msg}`);
    } finally {
      setUpdating(false);
    }
  };

  const hasChanges = 
    titleInput !== task.title ||
    statusInput !== normalizeStatus(task.status) ||
    priorityInput !== task.priority;

  return (
    <div className="text-sm border rounded-lg p-3 bg-gray-50 space-y-2">
      <div><span className="font-medium">ID:</span> {task.id}</div>
      <div><span className="font-medium">ProjectID:</span> {task.projectId}</div>
      <div className="flex items-center gap-2">
        <span className="font-medium">Title:</span>
        <input
          className="flex-1 border rounded px-2 py-1 text-sm"
          value={titleInput}
          onChange={(e) => setTitleInput(e.target.value)}
          disabled={updating}
        />
      </div>
      <div className="flex items-center gap-2">
        <span className="font-medium">Status:</span>
        <select
          className="flex-1 border rounded px-2 py-1 text-sm bg-white"
          value={statusInput}
          onChange={(e) => setStatusInput(e.target.value)}
          disabled={updating}
        >
          <option value="todo">todo</option>
          <option value="doing">doing</option>
          <option value="done">done</option>
        </select>
      </div>
      <div className="flex items-center gap-2">
        <span className="font-medium">Priority:</span>
        <select
          className="flex-1 border rounded px-2 py-1 text-sm bg-white"
          value={priorityInput}
          onChange={(e) => setPriorityInput(e.target.value)}
          disabled={updating}
        >
          <option value="low">low</option>
          <option value="medium">medium</option>
          <option value="high">high</option>
        </select>
      </div>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={handleUpdate}
          disabled={updating || !hasChanges || titleInput.trim() === ''}
          className="px-3 py-1 border rounded text-sm disabled:opacity-60 bg-white"
        >
          {updating ? 'Updating...' : 'Update'}
        </button>
      </div>
      {updateError && (
        <div className="text-xs text-red-600 whitespace-pre-wrap">
          Update Error: {updateError}
        </div>
      )}
    </div>
  );
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

  const [listLoading, setListLoading] = useState(false);
  const [listError, setListError] = useState<string | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);

  const projectIdForFetch = useMemo(() => projectId.trim(), [projectId]);

  const fetchTasks = async (pid: string) => {
    if (!pid) return;
    setListLoading(true);
    setListError(null);

    try {
      const res = await fetch(`/api/dev/tasks?projectId=${encodeURIComponent(pid)}`, {
        method: 'GET',
      });

      const text = await res.text();
      if (!res.ok) throw new Error(text);

      const data = JSON.parse(text) as Task[];
      setTasks(Array.isArray(data) ? data : []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setListError(msg);
      setTasks([]);
    } finally {
      setListLoading(false);
    }
  };

  // projectId が変わったら自動で一覧更新
  useEffect(() => {
    void fetchTasks(projectIdForFetch);
  }, [projectIdForFetch]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setCreateError(null);

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

      // 作成後に一覧を再取得（確実に全件が反映される）
      setId(generateTaskId());
      await fetchTasks(projectId.trim());
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setCreateError(msg);
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
          <p className="text-[11px] text-gray-500 mt-1">
            入力した Project ID のタスク一覧を自動で取得して表示します。
          </p>
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

        <div className="flex items-center gap-2">
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 border rounded text-sm disabled:opacity-60"
          >
            {loading ? 'Sending...' : 'Create Task'}
          </button>

          <button
            type="button"
            onClick={() => void fetchTasks(projectId.trim())}
            disabled={listLoading || !projectId.trim()}
            className="px-4 py-2 border rounded text-sm disabled:opacity-60"
          >
            {listLoading ? 'Loading...' : 'Refresh List'}
          </button>
        </div>
      </form>

      {createError && (
        <div className="text-sm text-red-600 whitespace-pre-wrap">
          Create Error: {createError}
        </div>
      )}

      {listError && (
        <div className="text-sm text-red-600 whitespace-pre-wrap">
          List Error: {listError}
        </div>
      )}

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">
            Tasks for <span className="font-mono">{projectIdForFetch || '(empty)'}</span>
          </h2>
          <span className="text-xs text-gray-500">
            {listLoading ? 'loading...' : `${tasks.length} item(s)`}
          </span>
        </div>

        {tasks.length === 0 && !listLoading && !listError && (
          <div className="text-sm text-gray-600 border rounded-lg p-3">
            タスクはありません（または projectId が不正です）
          </div>
        )}

        {tasks.map((t) => (
          <TaskRow
            key={t.id}
            task={t}
            onUpdateSuccess={() => void fetchTasks(projectIdForFetch)}
          />
        ))}
      </div>
    </div>
  );
}
