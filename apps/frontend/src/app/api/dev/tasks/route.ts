import { NextRequest, NextResponse } from 'next/server';

const TASKS_API_BASE_URL =
  process.env.TASKS_API_BASE_URL ?? 'http://localhost:8081/api';

export async function GET(req: NextRequest) {
  try {
    const projectId = req.nextUrl.searchParams.get('projectId') ?? '';
    if (!projectId) {
      return NextResponse.json({ message: 'projectId is required' }, { status: 400 });
    }

    const res = await fetch(
      `${TASKS_API_BASE_URL}/tasks?projectId=${encodeURIComponent(projectId)}`,
      { method: 'GET' },
    );

    const text = await res.text();

    // 404の場合は空配列を返す（タスクが0件の場合も404になる可能性があるため）
    if (res.status === 404) {
      return NextResponse.json([], { status: 200 });
    }

    try {
      const data = JSON.parse(text);
      return NextResponse.json(data, { status: res.status });
    } catch {
      return new NextResponse(text, { status: res.status });
    }
  } catch (err: unknown) {
    console.error('Error in GET /api/dev/tasks:', err);
    return NextResponse.json({ message: 'Internal error in dev proxy' }, { status: 500 });
  }
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();

    const res = await fetch(`${TASKS_API_BASE_URL}/tasks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    const text = await res.text();

    try {
      const data = JSON.parse(text);
      return NextResponse.json(data, { status: res.status });
    } catch {
      return new NextResponse(text, { status: res.status });
    }
  } catch (err: unknown) {
    console.error('Error in /api/dev/tasks:', err);
    return NextResponse.json(
      { message: 'Internal error in dev proxy' },
      { status: 500 },
    );
  }
}

export async function PATCH(req: NextRequest) {
  try {
    const id = req.nextUrl.searchParams.get('id');
    if (!id) {
      return NextResponse.json({ message: 'id is required' }, { status: 400 });
    }

    const body = await req.json();

    const res = await fetch(`${TASKS_API_BASE_URL}/tasks/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    const text = await res.text();

    try {
      const data = JSON.parse(text);
      return NextResponse.json(data, { status: res.status });
    } catch {
      return new NextResponse(text, { status: res.status });
    }
  } catch (err: unknown) {
    console.error('Error in PATCH /api/dev/tasks:', err);
    return NextResponse.json(
      { message: 'Internal error in dev proxy' },
      { status: 500 },
    );
  }
}
