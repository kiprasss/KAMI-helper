import { useEffect, useState } from "react";

import { OPACITY_EVENT, getGameStatus, isOpacityKey, loadOpacity } from "../overlay";
import type { GameStatus } from "../overlay";

const moscowTime = new Intl.DateTimeFormat("ru-RU", {
  timeZone: "Europe/Moscow",
  hour: "2-digit",
  minute: "2-digit",
});

function MiniOverlay() {
  const [now, setNow] = useState(() => new Date());
  const [status, setStatus] = useState<GameStatus | null>(null);
  const [opacity, setOpacity] = useState(() => loadOpacity("mini"));

  useEffect(() => {
    const refresh = () => setOpacity(loadOpacity("mini"));
    const onStorage = (event: StorageEvent) => {
      if (isOpacityKey(event.key)) refresh();
    };

    window.addEventListener("storage", onStorage);
    window.addEventListener(OPACITY_EVENT, refresh);

    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener(OPACITY_EVENT, refresh);
    };
  }, []);

  useEffect(() => {
    let stopped = false;

    const tick = async () => {
      setNow(new Date());

      try {
        const next = await getGameStatus();
        if (!stopped) setStatus(next);
      } catch {
        if (!stopped) setStatus(null);
      }
    };

    void tick();
    const timer = setInterval(tick, 1000);

    return () => {
      stopped = true;
      clearInterval(timer);
    };
  }, []);

  let gameText = "Игра не запущена";
  if (status?.running) {
    gameText = status.fps !== null ? `FPS ${status.fps}` : "FPS —";
  }

  return (
    <main
      className="flex h-full w-full items-center justify-center overflow-hidden select-none"
      style={{ opacity: opacity / 100 }}
    >
      <div
        className="
          flex items-center gap-2
          rounded-xl border px-3 py-2
          text-xs font-medium
          shadow-[0_8px_32px_rgba(0,0,0,0.18)]
          backdrop-blur-2xl
          border-amber-300/50 bg-amber-400/10 text-amber-300
        "
      >
        <span className="text-white">{moscowTime.format(now)}</span>
        <span className="text-white/35">·</span>
        <span className={status?.running ? "text-amber-300" : "text-zinc-300"}>
          {gameText}
        </span>
      </div>
    </main>
  );
}

export default MiniOverlay;
