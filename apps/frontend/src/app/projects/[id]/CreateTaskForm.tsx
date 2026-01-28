"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api/client";
import { normalizeApiError } from "@/lib/api/error";
import { ValidationIssues } from "@/components/ValidationIssues";
import type { ValidationIssue } from "@/lib/api/types";

// Client-side: use NEXT_PUBLIC_ prefix
const TASKS_BASE = process.env.NEXT_PUBLIC_TASKS_BASE ?? "http://localhost:8081/api";

type CreateTaskFormProps = {
  projectId: string;
};

export function CreateTaskForm({ projectId }: CreateTaskFormProps) {
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [priority, setPriority] = useState<"low" | "medium" | "high">("medium");
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
        method: "POST",
        body: JSON.stringify({
          title,
          priority,
          status: "todo",
        }),
      });

      // Success: reset title and refresh
      setTitle("");
      router.refresh();
    } catch (err) {
      const normalized = normalizeApiError(err);

      // ネットワーク系エラー（接続不可/サービス未起動）をユーザー向けに変換
      let userMessage = normalized.message;
      const isNetworkError =
        err instanceof TypeError ||
        normalized.message.toLowerCase().includes("failed to fetch") ||
        normalized.message.toLowerCase().includes("fetch failed") ||
        normalized.message.toLowerCase().includes("network");

      if (isNetworkError && !normalized.issues) {
        userMessage = "Tasksサービスに接続できません。起動しているか確認してください。";
      }

      setError({
        message: userMessage,
        issues: normalized.issues,
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-3 rounded-xl border bg-white p-4 shadow-sm">
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
            className="w-full rounded-lg border px-3 py-2 text-sm"
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
            onChange={(e) => setPriority(e.target.value as "low" | "medium" | "high")}
            className="w-full rounded-lg border px-3 py-2 text-sm"
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
          className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isSubmitting ? "作成中..." : "作成"}
        </button>
      </form>
    </div>
  );
}
