import { useEffect, useState } from "react";
import { FileText, Loader2, AlertCircle } from "lucide-react";
import { openUrl } from "@tauri-apps/plugin-opener";

import { imageSrc } from "../../api/client";
import { getRule } from "../../api/rules";
import type { RuleDetails } from "../../types/rules";
import type { SearchResult } from "../../types/search";

interface SearchResultDetailsProps {
  result: SearchResult | null;
}

function SearchResultDetails({ result }: SearchResultDetailsProps) {
  const [details, setDetails] = useState<RuleDetails | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!result) {
      setDetails(null);
      setError(null);
      setLoading(false);
      return;
    }

    let cancelled = false;

    setLoading(true);
    setError(null);
    setDetails(null);

    getRule(result.id)
      .then((data) => {
        if (!cancelled) {
          setDetails(data);
        }
      })
      .catch((err: unknown) => {
        console.error("Не удалось загрузить правило:", err);

        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Не удалось загрузить правило");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [result?.id]);

  if (!result) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
        <FileText size={28} className="text-zinc-600" />

        <p className="text-xs text-zinc-500">
          Выберите правило из списка
        </p>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center gap-2 text-xs text-zinc-400">
        <Loader2 size={17} className="animate-spin text-amber-400" />
        Загружаем правило...
      </div>
    );
  }

  if (error || !details) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 p-4 text-center">
        <AlertCircle size={24} className="text-red-400" />

        <p className="text-xs text-red-400">
          {error ?? "Правило не найдено"}
        </p>
      </div>
    );
  }

  const crumbs = details.breadcrumbs
    .split(" › ")
    .filter((part) => part && part !== "Главная" && part !== "Форум");
  const section = crumbs[crumbs.length - 1] ?? "";

  const openLink = (url: string) => {
    openUrl(url).catch((err: unknown) => {
      console.error("Не удалось открыть ссылку:", err);
    });
  };

  const openOnForum = () => openLink(details.url);

  const images = details.images ?? [];

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      <div className="shrink-0 border-b border-white/10 p-4">
        <div className="mb-3 flex items-center gap-2">
          <FileText size={14} className="text-amber-400" />

          <span className="text-[10px] font-semibold uppercase tracking-wider text-amber-400">
            {details.document_type === "project_rules"
              ? "Общие правила проекта"
              : "Законодательство"}
          </span>
        </div>

        <h2 className="text-sm font-semibold leading-5 text-white">
          {details.document_title}
        </h2>

        {section && (
          <p className="mt-2 text-xs text-zinc-500">
            {section}
          </p>
        )}

        <div className="mt-3 flex flex-wrap items-center gap-2">
          {details.point && (
            <span className="rounded-md border border-amber-400/30 bg-amber-400/10 px-2 py-1 text-[11px] font-medium text-amber-400">
              Правило {details.point}
            </span>
          )}

          {details.url && (
            <button
              type="button"
              onClick={openOnForum}
              title={details.url}
              className="rounded-md border border-white/10 bg-white/5 px-2 py-1 text-[11px] text-zinc-400"
            >
              Открыть на форуме
            </button>
          )}
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-4 select-text scrollbar-hide">
        <div className="space-y-4">
          <div>
            <p className="whitespace-pre-wrap text-xs leading-6 text-zinc-300">
              {details.text}
            </p>
          </div>

          {images.length > 0 && (
            <div className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
              <p className="mb-2 text-[10px] font-bold uppercase tracking-wider text-amber-400">
                Изображения
              </p>

              <div className="space-y-2">
                {images.map((url) => (
                  <img
                    key={url}
                    src={imageSrc(url)}
                    alt=""
                    loading="lazy"
                    draggable={false}
                    title="Открыть в браузере"
                    onClick={() => openLink(url)}
                    onError={(event) => {
                      event.currentTarget.style.display = "none";
                    }}
                    className="w-full cursor-pointer rounded-lg"
                  />
                ))}
              </div>
            </div>
          )}

          {details.punishment && (
            <div className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
              <p className="mb-2 text-[10px] font-bold uppercase tracking-wider text-amber-400">
                Наказание
              </p>

              <p className="whitespace-pre-wrap text-xs leading-6 text-zinc-300">
                {details.punishment}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default SearchResultDetails;
