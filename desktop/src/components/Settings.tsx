import { useEffect, useState } from "react";
import type { KeyboardEvent } from "react";
import { Blend, Keyboard, Monitor, PanelTop } from "lucide-react";

import { formatHotkey, parseHotkeyEvent } from "../hotkey";
import {
  listMonitors,
  loadMiniEnabled,
  loadMiniMonitor,
  loadOpacity,
  nextOpacity,
  saveMiniEnabled,
  saveMiniMonitor,
  saveOpacity,
} from "../overlay";
import type { MonitorInfo } from "../overlay";

const ROW_CLASS = `
  flex h-14 cursor-pointer items-center justify-between
  rounded-xl border px-4 outline-none transition-all
  border-white/10 bg-white/5 hover:border-amber-300/40
`;

const BADGE_CLASS =
  "rounded-lg border border-amber-300/30 bg-amber-300/10 px-3 py-1.5 text-sm font-semibold text-amber-300";

interface SettingsProps {
  hotkey: string;
  onHotkeyChange: (key: string) => void;
  isBinding: boolean;
  setIsBinding: (value: boolean) => void;
  error: string | null;
  onError: (message: string | null) => void;
}

function Settings({
  hotkey,
  onHotkeyChange,
  isBinding,
  setIsBinding,
  error,
  onError,
}: SettingsProps) {
  const [miniEnabled, setMiniEnabled] = useState(loadMiniEnabled);
  const [monitors, setMonitors] = useState<MonitorInfo[]>([]);
  const [monitorId, setMonitorId] = useState<string | null>(loadMiniMonitor);

  useEffect(() => {
    listMonitors()
      .then((list) => {
        setMonitors(list);

        if (!list.some((m) => m.id === monitorId)) {
          const fallback = list.find((m) => m.primary) ?? list[0];
          if (fallback) setMonitorId(fallback.id);
        }
      })
      .catch((err: unknown) => {
        console.error("Не удалось получить список мониторов:", err);
      });
  }, []);

  const [mainOpacity, setMainOpacity] = useState(() => loadOpacity("main"));
  const [miniOpacity, setMiniOpacity] = useState(() => loadOpacity("mini"));

  const cycleMainOpacity = () => {
    const next = nextOpacity(mainOpacity);
    setMainOpacity(next);
    saveOpacity("main", next);
  };

  const cycleMiniOpacity = () => {
    const next = nextOpacity(miniOpacity);
    setMiniOpacity(next);
    saveOpacity("mini", next);
  };

  const toggleMini = () => {
    const next = !miniEnabled;
    setMiniEnabled(next);
    saveMiniEnabled(next);
  };

  const nextMonitor = () => {
    if (monitors.length === 0) return;

    const index = monitors.findIndex((m) => m.id === monitorId);
    const next = monitors[(index + 1) % monitors.length];

    setMonitorId(next.id);
    saveMiniMonitor(next.id);
  };

  const activateOnKey = (action: () => void) => (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      action();
    }
  };

  const currentMonitor = monitors.find((m) => m.id === monitorId);

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!isBinding) {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        setIsBinding(true);
      }
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    const result = parseHotkeyEvent(event);

    switch (result.kind) {
      case "ignore":
        return;
      case "cancel":
        setIsBinding(false);
        return;
      case "invalid":
        onError(result.reason);
        return;
      case "ok":
        onError(null);
        onHotkeyChange(result.accelerator);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <div
        tabIndex={0}
        role="button"
        onClick={() => setIsBinding(true)}
        onBlur={() => {
          if (isBinding) setIsBinding(false);
        }}
        onKeyDown={handleKeyDown}
        className={`
          flex h-14 cursor-pointer items-center justify-between
          rounded-xl border px-4 outline-none transition-all
          ${
            isBinding
              ? "border-amber-300/70 bg-amber-300/10"
              : "border-white/10 bg-white/5 hover:border-amber-300/40"
          }
        `}
      >
        <div className="flex items-center gap-3">
          <Keyboard size={19} className="text-amber-400" />

          <span className="text-sm font-medium text-white">
            Клавиша открытия / скрытия
          </span>
        </div>

        <span className="rounded-lg border border-amber-300/30 bg-amber-300/10 px-3 py-1.5 text-sm font-semibold text-amber-300">
          {isBinding ? "Нажмите клавишу..." : formatHotkey(hotkey)}
        </span>
      </div>

      <p className="px-1 text-xs text-white/40">
        {error ??
          (isBinding
            ? "Нажмите нужную клавишу (F1–F24 или сочетание с Ctrl / Alt). ESC — отмена"
            : "Нажмите на блок, чтобы изменить клавишу открытия или скрытия")}
      </p>

      <div
        tabIndex={0}
        role="button"
        onClick={cycleMainOpacity}
        onKeyDown={activateOnKey(cycleMainOpacity)}
        className={ROW_CLASS}
      >
        <div className="flex items-center gap-3">
          <Blend size={19} className="text-amber-400" />

          <span className="text-sm font-medium text-white">
            Прозрачность главного окна
          </span>
        </div>

        <span className={BADGE_CLASS}>{mainOpacity}%</span>
      </div>

      <div
        tabIndex={0}
        role="button"
        onClick={toggleMini}
        onKeyDown={activateOnKey(toggleMini)}
        className={ROW_CLASS}
      >
        <div className="flex items-center gap-3">
          <PanelTop size={19} className="text-amber-400" />

          <span className="text-sm font-medium text-white">
            Мини-окно с временем и FPS
          </span>
        </div>

        <span className={BADGE_CLASS}>{miniEnabled ? "Вкл" : "Выкл"}</span>
      </div>

      {miniEnabled && (
        <div
          tabIndex={0}
          role="button"
          onClick={cycleMiniOpacity}
          onKeyDown={activateOnKey(cycleMiniOpacity)}
          className={ROW_CLASS}
        >
          <div className="flex items-center gap-3">
            <Blend size={19} className="text-amber-400" />

            <span className="text-sm font-medium text-white">
              Прозрачность мини-окна
            </span>
          </div>

          <span className={BADGE_CLASS}>{miniOpacity}%</span>
        </div>
      )}

      {miniEnabled && (
        <div
          tabIndex={0}
          role="button"
          onClick={nextMonitor}
          onKeyDown={activateOnKey(nextMonitor)}
          className={ROW_CLASS}
        >
          <div className="flex items-center gap-3">
            <Monitor size={19} className="text-amber-400" />

            <span className="text-sm font-medium text-white">
              Монитор для мини-окна
            </span>
          </div>

          <span className={BADGE_CLASS}>
            {currentMonitor?.label ?? "Основной"}
          </span>
        </div>
      )}

      <p className="px-1 text-xs text-white/40">
        Мини-окно (время по МСК и FPS) появляется сверху по центру выбранного монитора, когда вы скрываете окно клавишей. Нажимайте на прозрачность, чтобы менять её от 100% до 20%
      </p>
    </div>
  );
}

export default Settings;
