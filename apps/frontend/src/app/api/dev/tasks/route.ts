import { NextRequest, NextResponse } from 'next/server';

const TASKS_SERVICE_BASE = 'http://localhost:8081';

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();

    const res = await fetch(`${TASKS_SERVICE_BASE}/tasks`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    const text = await res.text();

    try {
      const data = JSON.parse(text);
      return NextResponse.json(data, { status: res.status });
    } catch {
      return new NextResponse(text, { status: res.status });
    }
  } catch (err: any) {
    console.error('Error in /api/dev/tasks:', err);
    return NextResponse.json(
      { message: 'Internal error in dev proxy' },
      { status: 500 },
    );
  }
}
