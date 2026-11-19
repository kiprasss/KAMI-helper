import { invoke } from "@tauri-apps/api/core";

const ENABLED_KEY = "kami-mini-enabled";
const MONITOR_KEY = "kami-mini-monitor";

export interface MonitorInfo {
  id: string;
  label: string;
  primary: boolean;
}

export interface GameStatus {
  running: boolean;
  fps: number | null;
  fps_available: boolean;
}

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    return;
  }
}

export function loadMiniEnabled(): boolean {
  return read(ENABLED_KEY) !== "0";
}

export function saveMiniEnabled(value: boolean) {
  write(ENABLED_KEY, value ? "1" : "0");
}

export function loadMiniMonitor(): string | null {
  return read(MONITOR_KEY);
}

export function saveMiniMonitor(id: string) {
  write(MONITOR_KEY, id);
}

export function listMonitors(): Promise<MonitorInfo[]> {
  return invoke<MonitorInfo[]>("list_monitors");
}

export async function showMini() {
  if (!loadMiniEnabled()) return;

  try {
    await invoke("mini_show", { monitor: loadMiniMonitor() });
  } catch (error) {
    console.error("Не удалось показать мини-окно:", error);
  }
}

export async function hideMini() {
  try {
    await invoke("mini_hide");
  } catch (error) {
    console.error("Не удалось скрыть мини-окно:", error);
  }
}

export function getGameStatus(): Promise<GameStatus> {
  return invoke<GameStatus>("game_status");
}

export type OpacityTarget = "main" | "mini";

export const OPACITY_EVENT = "kami-opacity";

const OPACITY_KEYS: Record<OpacityTarget, string> = {
  main: "kami-opacity-main",
  mini: "kami-opacity-mini",
};

export function loadOpacity(target: OpacityTarget): number {
  const value = Number(read(OPACITY_KEYS[target]));
  const saved = read(OPACITY_KEYS[target]);
  if (saved === null) return 90;
  return Number.isFinite(value) && value >= 20 && value <= 100 ? value : 90;
}

export function nextOpacity(current: number): number {
  return current <= 20 ? 100 : current - 10;
}

export function saveOpacity(target: OpacityTarget, value: number) {
  write(OPACITY_KEYS[target], String(value));
  window.dispatchEvent(new CustomEvent(OPACITY_EVENT));
}

export function isOpacityKey(key: string | null): boolean {
  return key === OPACITY_KEYS.main || key === OPACITY_KEYS.mini;
}
