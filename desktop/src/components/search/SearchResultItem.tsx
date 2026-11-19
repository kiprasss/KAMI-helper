import Highlight from "../Highlight";
import type { SearchResult } from "../../types/search";

interface SearchResultItemProps {
  result: SearchResult;
  selected: boolean;
  onClick: () => void;
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
            ? "border-accent-300/50 bg-accent-400/10"
            : "border-white/10 surface-glass hover:border-white/20 hover:bg-white/5"
        }
      `}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="truncate text-xs font-semibold text-white">
            {result.document_title}
          </h3>

          <p className="mt-1 line-clamp-2 text-[11px] leading-4 text-zinc-400">
            <Highlight text={result.snippet} />
          </p>
        </div>

        <span
          className={`
            shrink-0 rounded-md border px-2 py-1
            text-[10px] font-semibold
            ${
              selected
                ? "border-accent-300/30 bg-accent-400/10 text-accent-300"
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
