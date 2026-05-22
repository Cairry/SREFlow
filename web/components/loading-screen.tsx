"use client";

import { useEffect, useState } from "react";

export default function Loading() {
  const [progress, setProgress] = useState(0);
  const [fade, setFade] = useState(false);

  useEffect(() => {
    const startTime = Date.now();
    const duration = 900;

    const animate = () => {
      const elapsed = Date.now() - startTime;
      const p = Math.min(elapsed / duration, 1);
      setProgress(p);

      if (p < 1) {
        requestAnimationFrame(animate);
      } else {
        setTimeout(() => setFade(true), 50);
      }
    };

    requestAnimationFrame(animate);
  }, []);

  return (
    <div
      className={`fixed inset-0 z-[9999] flex flex-col items-center justify-center bg-black transition-opacity duration-300 ${
        fade ? "opacity-0 pointer-events-none" : "opacity-100"
      }`}
    >
      <div className="relative mb-8">
        <img
          src="/icon.png"
          alt="SREFlow AI"
          className="w-24 h-24 animate-pulse"
        />
      </div>

      <div className="text-white text-xl font-semibold tracking-wider mb-6 font-mono">
        SREFlow AI
      </div>
    </div>
  );
}
