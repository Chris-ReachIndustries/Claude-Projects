'use client';
import { createContext, useContext, useEffect, useRef, useState, useMemo, useCallback, type ReactNode } from 'react';

type SSEState = 'connecting' | 'connected' | 'disconnected';
type EventHandler = (data: any) => void;

interface SSEContextType {
  state: SSEState;
  subscribe: (event: string, handler: EventHandler) => () => void;
  reconnect: () => void;
}

const SSEContext = createContext<SSEContextType>({
  state: 'disconnected',
  subscribe: () => () => {},
  reconnect: () => {},
});

export function SSEProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<SSEState>('disconnected');
  const subscribersRef = useRef<Map<string, Set<EventHandler>>>(new Map());
  const sourceRef = useRef<EventSource | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  function connect() {
    const key = typeof window !== 'undefined' ? localStorage.getItem('cam_api_key') : null;
    if (!key) {
      setState('disconnected');
      reconnectTimerRef.current = setTimeout(connect, 2000);
      return;
    }

    sourceRef.current?.close();

    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9222';
    const url = `${apiUrl}/api/events?token=${key}`;

    setState('connecting');
    const source = new EventSource(url);
    sourceRef.current = source;

    source.onopen = () => setState('connected');
    source.onerror = () => {
      setState('disconnected');
      source.close();
      sourceRef.current = null;
      reconnectTimerRef.current = setTimeout(connect, 3000);
    };

    const eventTypes = [
      'agent-updated', 'agent-deleted', 'message-queued',
      'project-created', 'project-updated', 'project-deleted',
      'launch-request-created', 'launch-request-updated',
      'container-health', 'shutdown'
    ];

    eventTypes.forEach(type => {
      source.addEventListener(type, (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data);
          const handlers = subscribersRef.current.get(type);
          if (handlers) {
            handlers.forEach(h => h(data));
          }
        } catch {
          // ignore parse errors
        }
      });
    });
  }

  useEffect(() => {
    connect();
    return () => {
      sourceRef.current?.close();
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const subscribe = useCallback(function subscribe(event: string, handler: EventHandler): () => void {
    if (!subscribersRef.current.has(event)) {
      subscribersRef.current.set(event, new Set());
    }
    subscribersRef.current.get(event)!.add(handler);
    return () => {
      subscribersRef.current.get(event)?.delete(handler);
    };
  }, []);

  const reconnect = useCallback(function reconnect() {
    if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
    connect();
  }, []);

  const contextValue = useMemo(() => ({ state, subscribe, reconnect }), [state, subscribe, reconnect]);

  return (
    <SSEContext.Provider value={contextValue}>
      {children}
    </SSEContext.Provider>
  );
}

export function useSSE() { return useContext(SSEContext); }
