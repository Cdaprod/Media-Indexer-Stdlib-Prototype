// src/components/VideoSocketProvider.tsx
'use client';

import React, { createContext, useContext, ReactNode } from 'react';
import { useVideoSocket } from '@/lib/useVideoSocket';

interface Ctx {
  sendJSON: (payload: any) => void;
}

const VideoSocketCtx = createContext<Ctx | null>(null);

export function VideoSocketProvider({ children }: { children: ReactNode }) {
  const { sendJSON } = useVideoSocket(
    process.env.NEXT_PUBLIC_WS_BASE_URL ?? 'ws://localhost:8080/ws/camera',
    {
      onMessage: (ev) => {
        // whatever global dispatch you need
        console.log('[ws] message', ev.data);
      },
    },
  );

  return (
    <VideoSocketCtx.Provider value={{ sendJSON }}>
      {children}
    </VideoSocketCtx.Provider>
  );
}

export const useVideoSocketCtx = () => {
  const ctx = useContext(VideoSocketCtx);
  if (!ctx) throw new Error('useVideoSocketCtx must be within VideoSocketProvider');
  return ctx;
};