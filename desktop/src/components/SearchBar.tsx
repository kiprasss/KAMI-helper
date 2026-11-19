import { useState } from "react";
import type { FormEvent } from "react";
import { Search, X } from "lucide-react";

interface SearchBarProps {
  onSearch: (value: string) => void;
}

function SearchBar({ onSearch }: SearchBarProps) {
  const [query, setQuery] = useState("");

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const value = query.trim();

    if (!value) return;

    onSearch(value);
  };

  const clearSearch = () => {
    setQuery("");
    onSearch("");
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="
        group flex h-14 w-180 max-w-[90vw]
        items-center gap-3 rounded-2xl
        border border-accent-300/35
        surface-glass px-5
        shadow-[0_8px_32px_rgba(0,0,0,0.18)]

        transition-all duration-300
        focus-within:border-accent-300/80
        focus-within:shadow-[0_0_25px_var(--accent-glow)]
      "
    >
      <Search
        size={21}
        className="
          shrink-0 text-accent-400/65
          transition-colors
          group-focus-within:text-accent-300
        "
      />

      <input
        type="text"
        value={query}
        maxLength={200}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Введите запрос..."
        className="
          h-full min-w-0 flex-1
          bg-transparent text-sm font-medium
          text-white outline-none
          placeholder:text-white/35
        "
      />

      {query && (
        <button
          type="button"
          onClick={clearSearch}
          className="text-zinc-500 transition hover:text-white"
          title="Очистить поиск"
        >
          <X size={17} />
        </button>
      )}
    </form>
  );
}

export default SearchBar;
