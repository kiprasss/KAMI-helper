import { useEffect, useState } from "react";
import { Loader2, SearchX } from "lucide-react";

import type { SearchResult } from "../../types/search";
import type { KnowledgeStatus } from "../../types/status";
import SearchResultItem from "./SearchResultItem";
import SearchResultDetails from "./SearchResultDetails";

interface SearchResultsProps {
  results: SearchResult[];
  total: number;
  loading: boolean;
  error: string | null;
  query: string;
  status: KnowledgeStatus | null;
}

function SearchResults({
  results,
  total,
  loading,
  error,
  query,
  status,
}: SearchResultsProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);

  useEffect(() => {
    setSelectedId(results[0]?.id ?? null);
  }, [results]);

  if (!query) return null;

  const baseLoading = !loading && !error && results.length === 0 && status !== null && !status.ready;

  if (loading || error || results.length === 0) {
    return (
      <section
        className="
          flex min-h-40 w-full
          items-center justify-center
          rounded-2xl border border-white/10
          surface-panel p-6
          shadow-2xl
          select-none
        "
      >
        {loading || baseLoading ? (
          <div className="flex items-center gap-2 text-xs text-zinc-400">
            <Loader2 size={17} className="animate-spin text-accent-400" />
            {loading
              ? "Поиск..."
              : status?.blocked
                ? "Форум закрыт проверкой браузера, повторная попытка через 15 минут..."
                : "База правил загружается с форума..."}
          </div>
        ) : error ? (
          <p className="text-xs text-red-400">{error}</p>
        ) : (
          <div className="flex flex-col items-center gap-2 text-center">
            <SearchX size={26} className="text-zinc-500" />

            <p className="text-sm font-medium text-zinc-300">
              Ничего не найдено
            </p>

            <p className="text-xs text-zinc-500">
              Попробуйте изменить поисковый запрос
            </p>
          </div>
        )}
      </section>
    );
  }

  const selectedResult =
    results.find((item) => item.id === selectedId) ?? results[0];

  return (
    <section className="flex h-full min-h-0 w-full flex-col select-none">
      <div className="mb-2 flex shrink-0 items-center justify-between px-1">
        <span className="text-xs font-medium text-zinc-300">
          Результаты поиска
        </span>

        <span className="text-[11px] text-zinc-500">
          Найдено: {total}
        </span>
      </div>

      <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,0.85fr)_minmax(0,1.5fr)] gap-2">
        <div
          className="
            flex min-h-0 flex-col gap-2
            overflow-y-auto rounded-2xl
            border border-white/10
            surface-panel p-2
            shadow-2xl
            scrollbar-hide
          "
        >
          {results.map((result) => (
            <SearchResultItem
              key={result.id}
              result={result}
              selected={selectedResult?.id === result.id}
              onClick={() => setSelectedId(result.id)}
            />
          ))}
        </div>

        <div
          className="
            min-h-0 overflow-hidden rounded-2xl
            border border-white/10
            surface-panel
            shadow-2xl
          "
        >
          <SearchResultDetails result={selectedResult} />
        </div>
      </div>
    </section>
  );
}

export default SearchResults;
