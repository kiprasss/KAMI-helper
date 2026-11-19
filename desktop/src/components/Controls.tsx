import { useEffect, useState } from "react";
import { ChevronDown, Settings } from "lucide-react";

import { getServers } from "../api/servers";
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
  const [serverOpen, setServerOpen] = useState(false);
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
            onServerChange(data[0].code);
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

  const currentServer = servers.find(
    (item) => item.code === selectedServer
  );

  return (
    <div className="flex w-full items-center justify-between">
      <div className="relative">
        <button
          type="button"
          disabled={loading || servers.length === 0}
          onClick={() => setServerOpen(!serverOpen)}
          className="
            flex items-center gap-2
            rounded-xl border border-white/15
            bg-[#14141c]/40 px-3 py-2
            text-xs font-medium text-amber-400
            shadow-[0_8px_32px_rgba(0,0,0,0.18)]
            backdrop-blur-2xl
            transition-all duration-300
            hover:border-amber-300/50
            disabled:cursor-not-allowed disabled:opacity-50
          "
        >
          <span className="flex items-center gap-2 text-white">
            {currentServer && (
              <ServerIcon code={currentServer.code} />
            )}

            {loading
              ? "Загрузка..."
              : currentServer?.name ?? "Нет серверов"}
          </span>

          <ChevronDown
            size={14}
            className={`transition-transform ${
              serverOpen ? "rotate-180" : ""
            }`}
          />
        </button>

        {serverOpen && (
          <div
            className="
              absolute left-0 top-full z-50 mt-2
              min-w-full overflow-hidden rounded-lg
              border border-white/10
              bg-[#17171c] p-1 shadow-xl
            "
          >
            {servers.map((item) => (
              <button
                key={item.code}
                type="button"
                onClick={() => {
                  onServerChange(item.code);
                  setServerOpen(false);
                }}
                className={`
                  flex w-full items-center gap-2
                  rounded-md px-3 py-2
                  text-left text-xs transition
                  ${
                    selectedServer === item.code
                      ? "bg-amber-400/10 text-amber-300"
                      : "text-zinc-300 hover:bg-white/6"
                  }
                `}
              >
                <ServerIcon code={item.code} />
                {item.name}
              </button>
            ))}
          </div>
        )}
      </div>

      <button
        type="button"
        title="Настройки"
        onClick={() => setSettingsOpen(!settingsOpen)}
        className={`
          flex items-center gap-2
          rounded-xl border px-3 py-2
          text-xs font-medium
          shadow-[0_8px_32px_rgba(0,0,0,0.18)]
          backdrop-blur-2xl
          transition-all duration-300
          ${
            settingsOpen
              ? "border-amber-300/50 bg-amber-400/10 text-amber-300"
              : "border-white/15 bg-[#14141c]/40 text-zinc-400 hover:border-amber-300/50"
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
