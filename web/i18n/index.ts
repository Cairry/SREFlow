export const locales = ['en', 'zh'] as const;
export const defaultLocale = 'en' as const;

export type Locale = (typeof locales)[number];
export type Messages = typeof import('./messages/en.json');

// Simple translation function for client components
export function getNestedValue(obj: Record<string, unknown>, path: string): string {
  const keys = path.split('.');
  let result: unknown = obj;
  
  for (const key of keys) {
    if (result && typeof result === 'object' && key in result) {
      result = (result as Record<string, unknown>)[key];
    } else {
      return path; // Return path if not found
    }
  }
  
  return typeof result === 'string' ? result : path;
}

// Re-export from provider
export { useI18n, I18nProvider } from './provider';
