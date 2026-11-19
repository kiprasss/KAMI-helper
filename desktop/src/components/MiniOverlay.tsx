import { useEffect, useState } from "react";

import { getGameStatus } from "../overlay";
import type { GameStatus } from "../overlay";

const moscowTime = new Intl.DateTimeFormat("ru-RU", {
  timeZone: "Europe/Moscow",
  hour: "2-digit",
  minute: "2-digit",
});

function MiniOverlay() {
  const [now, setNow] = useState(() => new Date());
  const [status, setStatus] = useState<GameStatus | null>(null);

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
    <main className="flex h-full w-full items-center justify-center overflow-hidden select-none">
      <div
        className="
          flex items-center gap-2
          rounded-xl border border-white/15
          bg-[#14141c]/40 px-3 py-2
          text-xs font-medium text-amber-400
          shadow-[0_8px_32px_rgba(0,0,0,0.18)]
          backdrop-blur-2xl
        "
      >
        <span className="text-white">МСК {moscowTime.format(now)}</span>
        <span className="text-white/35">·</span>
        <span className={status?.running ? "text-amber-400" : "text-zinc-400"}>
          {gameText}
        </span>
      </div>
    </main>
  );
}

export default MiniOverlay;
