import { NextResponse } from 'next/server';
import { locales, type Locale } from '@/i18n';

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const { locale } = body as { locale: string };

    if (!locales.includes(locale as Locale)) {
      return NextResponse.json({ error: 'Invalid locale' }, { status: 400 });
    }

    const response = NextResponse.json({ success: true });
    response.cookies.set('locale', locale, {
      path: '/',
      maxAge: 60 * 60 * 24 * 365, // 1 year
      sameSite: 'lax',
    });

    return response;
  } catch (error) {
    return NextResponse.json({ error: 'Failed to set locale' }, { status: 500 });
  }
}

// GET endpoint to retrieve messages
export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const locale = searchParams.get('locale') || 'en';

  if (!locales.includes(locale as Locale)) {
    return NextResponse.json({ error: 'Invalid locale' }, { status: 400 });
  }

  try {
    const messages = (await import(`@/i18n/messages/${locale}.json`)).default;
    return NextResponse.json({ messages });
  } catch (error) {
    return NextResponse.json({ error: 'Failed to load messages' }, { status: 500 });
  }
}
