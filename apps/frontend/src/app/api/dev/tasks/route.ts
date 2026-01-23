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

    // 404の処理: projectIdが指定されている場合のみ、旧Tasks APIの0件が404になるケースを考慮
    // ただし、本当の404（パスミス/サービス未起動/ルーティング崩れ）はそのまま返す
    if (res.status === 404) {
      // projectIdが指定されていて、かつ tasks list API へのリクエストの場合のみ
      // 0件の可能性があるため空配列を返す
      // それ以外の404はそのまま返す（エラーとして検知可能にする）
      const requestUrl = `${TASKS_API_BASE_URL}/tasks?projectId=${encodeURIComponent(projectId)}`;
      if (projectId && requestUrl.includes('/tasks')) {
        // レスポンスボディが空の場合は0件とみなす
        // エラーメッセージがある場合は本当の404として扱う
        if (!text || text.trim() === '') {
          return NextResponse.json([], { status: 200 });
        }
        // エラーメッセージがある場合は本当の404として返す
        console.warn(
          `GET /api/dev/tasks: upstream returned 404 with body for projectId=${projectId}: ${text}`
        );
      }
      // 404をそのまま返す
      try {
        const data = JSON.parse(text);
        return NextResponse.json(data, { status: 404 });
      } catch {
        return new NextResponse(text, { status: 404 });
      }
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
