import type { KeyboardEvent } from "react";

import { getCurrentWindow } from "@tauri-apps/api/window";

import { hideMini, showMini } from "./overlay";
import {
  register,
  unregister,
} from "@tauri-apps/plugin-global-shortcut";

export const DEFAULT_HOTKEY = "F4";
export const HOTKEY_STORAGE_KEY = "kami-hotkey";

const MODIFIER_CODES = new Set([
  "ControlLeft",
  "ControlRight",
  "ShiftLeft",
  "ShiftRight",
  "AltLeft",
  "AltRight",
  "MetaLeft",
  "MetaRight",
  "OSLeft",
  "OSRight",
]);

const STANDALONE_KEYS =
  /^(F([1-9]|1[0-9]|2[0-4])|Insert|Pause|ScrollLock|Home|End|PageUp|PageDown)$/;

export type HotkeyParseResult =
  | { kind: "ignore" }
  | { kind: "cancel" }
  | { kind: "invalid"; reason: string }
  | { kind: "ok"; accelerator: string };

export function parseHotkeyEvent(event: KeyboardEvent): HotkeyParseResult {
  const { code } = event;

  if (code === "Escape") return { kind: "cancel" };
  if (!code || MODIFIER_CODES.has(code)) return { kind: "ignore" };

  const modifiers: string[] = [];
  if (event.ctrlKey) modifiers.push("Ctrl");
  if (event.altKey) modifiers.push("Alt");
  if (event.shiftKey) modifiers.push("Shift");
  if (event.metaKey) modifiers.push("Super");

  const hasRealModifier = event.ctrlKey || event.altKey || event.metaKey;

  if (!hasRealModifier && !STANDALONE_KEYS.test(code)) {
    return {
      kind: "invalid",
      reason:
        "Эту клавишу нельзя назначить без Ctrl или Alt — она будет мешать набору текста. Используйте F1–F24 или сочетание, например Ctrl+Shift+K.",
    };
  }

  return { kind: "ok", accelerator: [...modifiers, code].join("+") };
}

export function formatHotkey(accelerator: string): string {
  return accelerator
    .split("+")
    .map((part) =>
      part
        .replace(/^Key([A-Z])$/, "$1")
        .replace(/^Digit(\d)$/, "$1")
        .replace(/^Numpad(\w+)$/, "Num $1")
    )
    .join(" + ");
}

async function toggleMainWindow() {
  try {
    const window = getCurrentWindow();

    if (await window.isVisible()) {
      await window.hide();
      await showMini();
    } else {
      await hideMini();
      await window.show();
      await window.setFocus();
    }
  } catch (error) {
    console.error("Ошибка переключения окна:", error);
  }
}

export async function registerToggleHotkey(
  accelerator: string,
  isPaused: () => boolean
) {
  await register(accelerator, (event) => {
    if (event.state !== "Pressed") return;
    if (isPaused()) return;

    void toggleMainWindow();
  });
}

export async function unregisterHotkey(accelerator: string) {
  await unregister(accelerator);
}

export function loadSavedHotkey(): string | null {
  let saved: string | null = null;

  try {
    saved = localStorage.getItem(HOTKEY_STORAGE_KEY);
  } catch {
    return null;
  }

  if (!saved) return null;

  if (saved.includes("+") || STANDALONE_KEYS.test(saved)) {
    return saved;
  }

  return null;
}

export function saveHotkey(accelerator: string) {
  try {
    localStorage.setItem(HOTKEY_STORAGE_KEY, accelerator);
  } catch {
    return;
  }
}
