import type { SearchResult } from "../../types/search";

interface SearchResultItemProps {
  result: SearchResult;
  selected: boolean;
  onClick: () => void;
}

function plainSnippet(snippet: string) {
  return snippet.replace(/<\/?mark>/g, "");
}

function SearchResultItem({
  result,
  selected,
  onClick,
}: SearchResultItemProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`
        w-full shrink-0 rounded-xl border p-3 text-left
        transition-all duration-200
        ${
          selected
            ? "border-amber-300/50 bg-amber-400/10"
            : "border-white/10 bg-[#14141c]/50 hover:border-white/20 hover:bg-white/5"
        }
      `}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="truncate text-xs font-semibold text-white">
            {result.document_title}
          </h3>

          <p className="mt-1 truncate text-[11px] text-zinc-500">
            {plainSnippet(result.snippet)}
          </p>
        </div>

        <span
          className={`
            shrink-0 rounded-md border px-2 py-1
            text-[10px] font-semibold
            ${
              selected
                ? "border-amber-300/30 bg-amber-400/10 text-amber-300"
                : "border-white/10 bg-white/5 text-zinc-400"
            }
          `}
        >
          {result.point || "Текст"}
        </span>
      </div>
    </button>
  );
}

export default SearchResultItem;
