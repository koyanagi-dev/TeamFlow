import { NextRequest, NextResponse } from 'next/server';

const PROJECTS_SERVICE_BASE = 'http://localhost:8080';

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();

    const res = await fetch(`${PROJECTS_SERVICE_BASE}/projects`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      // そのまま透過
      body: JSON.stringify(body),
    });

    const text = await res.text();

    // Go 側のレスポンスが JSON 想定なので、できれば JSON として返す
    try {
      const data = JSON.parse(text);
      return NextResponse.json(data, { status: res.status });
    } catch {
      // JSON でなければそのままテキストで返す
      return new NextResponse(text, { status: res.status });
    }
  } catch (err: any) {
    console.error('Error in /api/dev/projects:', err);
    return NextResponse.json(
      { message: 'Internal error in dev proxy' },
      { status: 500 },
    );
  }
}
