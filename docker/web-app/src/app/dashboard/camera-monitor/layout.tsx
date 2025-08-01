'use client';
import Link from 'next/link';
import type { ReactNode } from 'react';

export default function CameraMonitorLayout({ children }: { children: ReactNode }) {
  return (
    <div className="w-screen h-screen bg-black flex flex-col">
      {/* Back button overlays top-left */}
      <div className="absolute top-4 left-4 z-10">
        <Link
          href="/dashboard"
          className="text-white bg-gray-800 px-3 py-1 rounded hover:bg-gray-700 transition"
        >
          ← Back
        </Link>
      </div>

      {/* Camera UI fills the rest */}
      <div className="flex-1 overflow-hidden">{children}</div>
    </div>
  );
}