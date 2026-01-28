import { NextResponse } from "next/server";

const PROJECTS_SERVICE_BASE = "http://localhost:8080";

export async function GET() {
  try {
    const res = await fetch(`${PROJECTS_SERVICE_BASE}/projects`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
    });

    const text = await res.text();

    try {
      const json = JSON.parse(text);
      return NextResponse.json(json, { status: res.status });
    } catch {
      return new NextResponse(text, { status: res.status });
    }
  } catch (err) {
    console.error("Error in GET /api/dev/projects/list:", err);
    return NextResponse.json({ message: "internal error in dev proxy" }, { status: 500 });
  }
}
