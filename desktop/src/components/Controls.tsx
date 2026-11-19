import { useEffect, useState } from "react";
import { Settings } from "lucide-react";

import { getServers } from "../api/servers";
import { loadServer } from "../theme";
import type { Server } from "../types/server";

const SERVER_ICONS: Record<string, string> = {
  MIRAGE: "https://api.novarp.games/images/servers/mirage.svg",
  NEW_ERA: "https://api.novarp.games/images/servers/new-era.svg",
  HORIZON:
    "https://s3.grandroleplay.ru/grand-assets/server-icons/1fb4f1f4e0fd7f2bff945360d6da8198f10fb75b5b9e8086fdf42b38e8e143da.svg",
};

function ServerIcon({ code }: { code: string }) {
  const [failed, setFailed] = useState(false);
  const src = SERVER_ICONS[code];

  if (!src || failed) return null;

  return (
    <img
      src={src}
      alt=""
      className="h-5 w-5 shrink-0 object-contain"
      draggable={false}
      onError={() => setFailed(true)}
    />
  );
}

interface ControlsProps {
  settingsOpen: boolean;
  setSettingsOpen: (open: boolean) => void;
  selectedServer: string;
  onServerChange: (server: string) => void;
}

function Controls({
  settingsOpen,
  setSettingsOpen,
  selectedServer,
  onServerChange,
}: ControlsProps) {
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    let retry: ReturnType<typeof setTimeout> | undefined;

    const load = () => {
      getServers()
        .then((data) => {
          if (!mounted) return;

          setServers(data);
          setLoading(false);

          if (data.length > 0) {
            const saved = loadServer();
            const initial = data.find((item) => item.code === saved) ?? data[0];
            onServerChange(initial.code);
          }
        })
        .catch((error) => {
          console.error("Не удалось загрузить список серверов:", error);
          if (mounted) retry = setTimeout(load, 5_000);
        });
    };

    load();

    return () => {
      mounted = false;
      if (retry) clearTimeout(retry);
    };
  }, []);

  return (
    <div className="flex w-full items-center justify-between gap-2">
      <div className="flex min-w-0 items-center gap-1 rounded-xl border border-white/10 surface-glass p-1">
        {loading && (
          <span className="px-3 py-1.5 text-xs text-zinc-400">Загрузка серверов...</span>
        )}

        {servers.map((item) => (
          <button
            key={item.code}
            type="button"
            onClick={() => onServerChange(item.code)}
            className={`
              flex items-center gap-2
              rounded-lg border px-3 py-1.5
              text-xs font-medium transition-colors duration-200
              ${
                selectedServer === item.code
                  ? "border-accent-300/50 bg-accent-400/10 text-white"
                  : "border-transparent text-zinc-400 hover:bg-white/5 hover:text-zinc-200"
              }
            `}
          >
            <ServerIcon code={item.code} />
            {item.name}
          </button>
        ))}
      </div>

      <button
        type="button"
        title="Настройки"
        onClick={() => setSettingsOpen(!settingsOpen)}
        className={`
          flex shrink-0 items-center gap-2
          rounded-xl border px-3 py-2
          text-xs font-medium
          transition-colors duration-200
          ${
            settingsOpen
              ? "border-accent-300/50 bg-accent-400/10 text-accent-300"
              : "border-white/10 surface-glass text-zinc-400 hover:border-accent-300/50"
          }
        `}
      >
        <Settings size={16} />
        <span className="text-white">Настройки</span>
      </button>
    </div>
  );
}

export default Controls;
