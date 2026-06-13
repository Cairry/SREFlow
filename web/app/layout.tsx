import React from "react"
import type { Metadata } from 'next'
import { cookies } from 'next/headers'
import { ThemeProvider } from "@/components/theme-provider"
import { Navigation } from "@/components/navigation"
import { Toaster } from "@/components/ui/toaster"
import { I18nProvider, defaultLocale, type Locale, type Messages } from "@/i18n"

import './globals.css'

export const metadata: Metadata = {
  title: 'SREFlow AI',
  icons: {
    icon: '/icon.png',
  },
}

import Loading from "@/components/loading-screen"

// Server-side function to get locale from cookie
async function getLocale(): Promise<Locale> {
  const cookieStore = await cookies();
  const localeCookie = cookieStore.get('locale');
  const locales: Locale[] = ['en', 'zh'];
  
  return localeCookie?.value && locales.includes(localeCookie.value as Locale)
    ? localeCookie.value as Locale
    : defaultLocale;
}

// Server-side function to load messages
async function getMessages(locale: Locale): Promise<Messages> {
  try {
    const messages = (await import(`@/i18n/messages/${locale}.json`)).default;
    return messages as Messages;
  } catch (error) {
    // Fallback to English if loading fails
    const messages = (await import(`@/i18n/messages/en.json`)).default;
    return messages as Messages;
  }
}

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  const locale = await getLocale();
  const messages = await getMessages(locale);

  return (
    <html lang={locale} suppressHydrationWarning>
      <body className="font-sans antialiased" suppressHydrationWarning>
        <I18nProvider initialLocale={locale} initialMessages={messages}>
          <ThemeProvider
            attribute="class"
            defaultTheme="dark"
            enableSystem={false}
            storageKey="flowforge-theme"
          >
            <Loading />
            <Navigation />
            {children}
            <Toaster />
          </ThemeProvider>
        </I18nProvider>
      </body>
    </html>
  )
}
