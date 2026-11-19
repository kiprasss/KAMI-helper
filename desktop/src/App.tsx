import { useEffect, useRef, useState } from "react";
import type { MouseEvent } from "react";

import { getCurrentWindow } from "@tauri-apps/api/window";

import SearchBar from "./components/SearchBar";
import Controls from "./components/Controls";
import Settings from "./components/Settings";
import SearchResults from "./components/search/SearchResults";

import { searchRules } from "./api/search";
import { getStatus } from "./api/status";
import type { SearchResult } from "./types/search";
import type { KnowledgeStatus } from "./types/status";
import {
  DEFAULT_HOTKEY,
  loadSavedHotkey,
  registerToggleHotkey,
  saveHotkey,
  unregisterHotkey,
} from "./hotkey";

const savedHotkey = loadSavedHotkey();

function App() {
  useEffect(() => {
    getCurrentWindow().setAlwaysOnTop(true).catch(console.error);
  }, []);

  const [settingsOpen, setSettingsOpen] = useState(savedHotkey === null);
  const [isBinding, setIsBinding] = useState(false);
  const [hotkeyError, setHotkeyError] = useState<string | null>(null);

  const [hotkey, setHotkey] = useState(savedHotkey ?? DEFAULT_HOTKEY);

  const [selectedServer, setSelectedServer] = useState("");
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [total, setTotal] = useState(0);
  const [searchLoading, setSearchLoading] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);

  const [status, setStatus] = useState<KnowledgeStatus | null>(null);

  const registeredHotkey = useRef<string | null>(null);
  const isBindingRef = useRef(false);
  const searchRequestId = useRef(0);

  const isPaused = () => isBindingRef.current;

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    let stopped = false;

    const poll = async () => {
      let delay = 60_000;

      try {
        const next = await getStatus();
        if (stopped) return;

        setStatus(next);

        if (!next.ready || next.refreshing) delay = 3_000;
      } catch {
        if (stopped) return;

        delay = 5_000;
      }

      timer = setTimeout(poll, delay);
    };

    void poll();

    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
    };
  }, []);

  useEffect(() => {
    let mounted = true;
    const initialHotkey = hotkey;

    registerToggleHotkey(initialHotkey, isPaused)
      .then(async () => {
        if (!mounted) {
          await unregisterHotkey(initialHotkey);
          return;
        }

        registeredHotkey.current = initialHotkey;
      })
      .catch((error) => {
        console.error("Ошибка регистрации горячей клавиши:", error);

        if (mounted) {
          setSettingsOpen(true);
          setHotkeyError(
            `Не удалось зарегистрировать клавишу ${initialHotkey}. Возможно, она занята другой программой — выберите другую.`
          );
        }
      });

    return () => {
      mounted = false;

      if (registeredHotkey.current) {
        unregisterHotkey(registeredHotkey.current).catch(() => {});
        registeredHotkey.current = null;
      }
    };
  }, []);

  const changeBindingState = (value: boolean) => {
    isBindingRef.current = value;
    setIsBinding(value);

    if (value) setHotkeyError(null);
  };

  const changeHotkey = async (key: string) => {
    if (key.toLowerCase() === registeredHotkey.current?.toLowerCase()) {
      changeBindingState(false);
      return;
    }

    const previousHotkey = registeredHotkey.current;

    try {
      if (previousHotkey) {
        await unregisterHotkey(previousHotkey);
        registeredHotkey.current = null;
      }

      await registerToggleHotkey(key, isPaused);

      registeredHotkey.current = key;
      setHotkey(key);
      saveHotkey(key);
      changeBindingState(false);
    } catch (error) {
      console.error("Ошибка смены клавиши:", error);

      if (previousHotkey && registeredHotkey.current === null) {
        try {
          await registerToggleHotkey(previousHotkey, isPaused);
          registeredHotkey.current = previousHotkey;
        } catch (restoreError) {
          console.error("Не удалось вернуть прежнюю клавишу:", restoreError);
        }
      }

      changeBindingState(false);
      setHotkeyError(
        "Не удалось назначить эту клавишу. Возможно, она занята другой программой."
      );
    }
  };

  const runSearch = async (searchQuery: string, server: string) => {
    const normalizedQuery = searchQuery.trim();

    if (!normalizedQuery || !server) return;

    const requestId = ++searchRequestId.current;

    setSearchLoading(true);
    setSearchError(null);

    try {
      const response = await searchRules({
        query: normalizedQuery,
        server,
      });

      if (requestId !== searchRequestId.current) return;

      setResults(response.data);
      setTotal(response.meta.total);
    } catch (error) {
      if (requestId !== searchRequestId.current) return;

      console.error("Search error:", error);
      setResults([]);
      setTotal(0);
      setSearchError("Не удалось выполнить поиск. Проверьте подключение к серверу.");
    } finally {
      if (requestId === searchRequestId.current) {
        setSearchLoading(false);
      }
    }
  };

  useEffect(() => {
    if (status?.ready && query.trim() && results.length === 0) {
      runSearch(query, selectedServer);
    }
  }, [status?.ready]);

  const startDragging = async (event: MouseEvent) => {
    if (event.button !== 0) return;

    if (
      (event.target as HTMLElement).closest(
        "button, input, textarea, select, a, [tabindex]"
      )
    ) {
      return;
    }

    try {
      await getCurrentWindow().startDragging();
    } catch (error) {
      console.error("Ошибка перемещения окна:", error);
    }
  };

  const handleSearch = (value: string) => {
    setQuery(value);

    if (!value.trim()) {
      searchRequestId.current++;
      setResults([]);
      setTotal(0);
      setSearchError(null);
      setSearchLoading(false);
      return;
    }

    runSearch(value, selectedServer);
  };

  const handleServerChange = (server: string) => {
    setSelectedServer(server);

    if (query.trim()) {
      runSearch(query, server);
    }
  };

  return (
    <main className="h-full w-full overflow-hidden" onMouseDown={startDragging}>
      <div className="flex h-full w-full flex-col items-center gap-3 overflow-hidden p-3">
        <div className="flex w-full max-w-[744px] shrink-0 flex-col gap-2 rounded-xl bg-black/30 p-3">
          <SearchBar onSearch={handleSearch} />

          <Controls
            settingsOpen={settingsOpen}
            setSettingsOpen={setSettingsOpen}
            selectedServer={selectedServer}
            onServerChange={handleServerChange}
          />

          {settingsOpen && (
            <div className="w-full rounded-xl bg-black/20 p-3">
              <Settings
                hotkey={hotkey}
                onHotkeyChange={changeHotkey}
                isBinding={isBinding}
                setIsBinding={changeBindingState}
                error={hotkeyError}
                onError={setHotkeyError}
              />
            </div>
          )}
        </div>

        <div className="w-full max-w-[744px] min-h-0 flex-1">
          <SearchResults
            results={results}
            total={total}
            loading={searchLoading}
            error={searchError}
            query={query}
            status={status}
          />
        </div>
      </div>
    </main>
  );
}

export default App;
