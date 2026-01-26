'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { apiFetch } from '@/lib/api/client';
import { normalizeApiError } from '@/lib/api/error';
import { ValidationIssues } from '@/components/ValidationIssues';
import type { ValidationIssue } from '@/lib/api/types';

// Client-side: use NEXT_PUBLIC_ prefix
const TASKS_BASE =
  process.env.NEXT_PUBLIC_TASKS_BASE ?? 'http://localhost:8081/api';

type CreateTaskFormProps = {
  projectId: string;
};

export function CreateTaskForm({ projectId }: CreateTaskFormProps) {
  const router = useRouter();
  const [title, setTitle] = useState('');
  const [priority, setPriority] = useState<'low' | 'medium' | 'high'>('medium');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<{
    message: string;
    issues?: ValidationIssue[];
  } | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      await apiFetch(`${TASKS_BASE}/projects/${projectId}/tasks`, {
        method: 'POST',
        body: JSON.stringify({
          title,
          priority,
          status: 'todo',
        }),
      });

      // Success: reset title and refresh
      setTitle('');
      router.refresh();
    } catch (err) {
      const normalized = normalizeApiError(err);
      setError({
        message: normalized.message,
        issues: normalized.issues,
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="border rounded-xl p-4 bg-white shadow-sm space-y-3">
      <h3 className="text-lg font-semibold">新しいタスクを作成</h3>
      <form onSubmit={handleSubmit} className="space-y-3">
        <div className="space-y-1">
          <label htmlFor="title" className="block text-sm font-medium">
            Title <span className="text-red-600">*</span>
          </label>
          <input
            id="title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full border rounded-lg px-3 py-2 text-sm"
            placeholder="タスクのタイトルを入力"
            disabled={isSubmitting}
          />
        </div>

        <div className="space-y-1">
          <label htmlFor="priority" className="block text-sm font-medium">
            Priority
          </label>
          <select
            id="priority"
            value={priority}
            onChange={(e) =>
              setPriority(e.target.value as 'low' | 'medium' | 'high')
            }
            className="w-full border rounded-lg px-3 py-2 text-sm"
            disabled={isSubmitting}
          >
            <option value="low">low</option>
            <option value="medium">medium</option>
            <option value="high">high</option>
          </select>
        </div>

        {error && (
          <div className="space-y-2">
            <p className="text-sm text-red-600">{error.message}</p>
            <ValidationIssues issues={error.issues} />
          </div>
        )}

        <button
          type="submit"
          disabled={isSubmitting}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isSubmitting ? '作成中...' : '作成'}
        </button>
      </form>
    </div>
  );
}
