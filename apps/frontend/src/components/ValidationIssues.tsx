import type { ValidationIssue } from "@/lib/api/types";

export function ValidationIssues({ issues }: { issues?: ValidationIssue[] }) {
  if (!issues?.length) return null;
  return (
    <div className="rounded-lg border bg-white p-3 text-sm">
      <div className="mb-2 font-semibold text-red-700">Validation errors</div>
      <ul className="space-y-1">
        {issues.map((i, idx) => (
          <li key={idx} className="text-red-700">
            <span className="font-mono">
              {i.location ?? "body"}.{i.field}
            </span>{" "}
            — {i.code}: {i.message}
            {typeof i.rejectedValue !== "undefined" && (
              <span className="text-gray-600"> (rejected: {String(i.rejectedValue)})</span>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
