import { useEffect, useState } from "react";

import { getGameStatus, loadOpacity } from "../overlay";
import type { GameStatus } from "../overlay";
import { applySurfaceAlpha, applyTheme, loadServer } from "../theme";

const moscowTime = new Intl.DateTimeFormat("ru-RU", {
  timeZone: "Europe/Moscow",
  hour: "2-digit",
  minute: "2-digit",
});

function syncAppearance() {
  applyTheme(loadServer());
  applySurfaceAlpha(loadOpacity("mini"));
}

function MiniOverlay() {
  const [now, setNow] = useState(() => new Date());
  const [status, setStatus] = useState<GameStatus | null>(null);

  useEffect(() => {
    syncAppearance();

    window.addEventListener("storage", syncAppearance);

    let stopped = false;

    const tick = async () => {
      if (document.hidden) return;

      setNow(new Date());
      syncAppearance();

      try {
        const next = await getGameStatus();
        if (!stopped) setStatus(next);
      } catch {
        if (!stopped) setStatus(null);
      }
    };

    void tick();
    const timer = setInterval(tick, 1000);
    document.addEventListener("visibilitychange", tick);

    return () => {
      stopped = true;
      clearInterval(timer);
      window.removeEventListener("storage", syncAppearance);
      document.removeEventListener("visibilitychange", tick);
    };
  }, []);

  let gameText = "Игра не запущена";
  if (status?.running) {
    gameText = status.fps !== null ? `${status.fps} FPS` : "FPS —";
  }

  return (
    <main className="flex h-full w-full items-center justify-center overflow-hidden select-none">
      <div
        className="
          surface-mini flex items-center gap-2
          rounded-xl border border-accent-300/50 px-3.5 py-1.5
          text-xs font-semibold tabular-nums
          shadow-[0_4px_16px_rgba(0,0,0,0.25)]
        "
      >
        <span className="text-white">{moscowTime.format(now)}</span>
        <span className="h-3 w-px bg-accent-300/40" />
        <span className={status?.running ? "text-accent-300" : "text-zinc-300"}>
          {gameText}
        </span>
      </div>
    </main>
  );
}

export default MiniOverlay;
