import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import { SSEProvider } from '@/providers/sse-provider';
import { Sidebar } from '@/components/sidebar';
import { AuthGate } from '@/components/auth-gate';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'Claude Projects',
  description: 'AI Agent Orchestration Platform',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="light" style={{ colorScheme: 'light' }}>
      <body className={inter.className}>
        <AuthGate>
          <SSEProvider>
            <div className="flex h-screen">
              <Sidebar />
              <main className="flex-1 overflow-auto bg-background">
                {children}
              </main>
            </div>
          </SSEProvider>
        </AuthGate>
      </body>
    </html>
  );
}
