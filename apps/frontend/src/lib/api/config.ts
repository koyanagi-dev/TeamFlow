/**
 * API base URLs. Centralize here for future /api/projects etc. migration.
 * Projects service: no /api prefix. Tasks service: under /api.
 */
export const PROJECTS_BASE = process.env.NEXT_PUBLIC_PROJECTS_BASE ?? "http://localhost:8080";
export const TASKS_BASE = process.env.NEXT_PUBLIC_TASKS_BASE ?? "http://localhost:8081/api";
