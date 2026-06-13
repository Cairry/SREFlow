'use client';

import { createContext, useContext, type ReactNode, useState, useEffect } from 'react';
import { locales, defaultLocale, type Locale, type Messages, getNestedValue } from './index';

interface I18nContextType {
  locale: Locale;
  setLocale: (locale: Locale) => Promise<void>;
  messages: Messages;
  t: (key: string, params?: Record<string, string>) => string;
}

const I18nContext = createContext<I18nContextType | null>(null);

export function useI18n() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error('useI18n must be used within an I18nProvider');
  }
  return context;
}

interface I18nProviderProps {
  children: ReactNode;
  initialLocale?: Locale;
  initialMessages?: Messages;
}

export function I18nProvider({ children, initialLocale, initialMessages }: I18nProviderProps) {
  const [locale, setLocaleState] = useState<Locale>(initialLocale || defaultLocale);
  const [messages, setMessages] = useState<Messages>(initialMessages || {});

  // Load initial messages
  useEffect(() => {
    if (!initialMessages) {
      loadMessages(locale);
    }
  }, []);

  const loadMessages = async (newLocale: Locale) => {
    try {
      const response = await fetch(`/api/locale/messages?locale=${newLocale}`);
      if (response.ok) {
        const data = await response.json();
        setMessages(data.messages);
      }
    } catch (error) {
      console.error('Failed to load messages:', error);
    }
  };

  const setLocale = async (newLocale: Locale) => {
    try {
      // Set cookie via API
      await fetch('/api/locale', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ locale: newLocale }),
      });
      
      // Load new messages
      await loadMessages(newLocale);
      setLocaleState(newLocale);
      
      // Reload page to apply changes
      window.location.reload();
    } catch (error) {
      console.error('Failed to change locale:', error);
    }
  };

  // Translation function with parameter substitution
  const t = (key: string, params?: Record<string, string>): string => {
    let text = getNestedValue(messages as unknown as Record<string, unknown>, key);
    
    if (params) {
      Object.entries(params).forEach(([paramKey, value]) => {
        text = text.replace(new RegExp(`{${paramKey}}`, 'g'), value);
      });
    }
    
    return text;
  };

  return (
    <I18nContext.Provider value={{ locale, setLocale, messages, t }}>
      {children}
    </I18nContext.Provider>
  );
}
