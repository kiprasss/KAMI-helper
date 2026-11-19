import { Fragment } from "react";

interface HighlightProps {
  text: string;
}

function Highlight({ text }: HighlightProps) {
  return (
    <>
      {text.split(/(<mark>.*?<\/mark>)/g).map((part, index) => {
        const match = part.match(/^<mark>(.*)<\/mark>$/);

        if (match) {
          return (
            <mark key={index} className="rounded bg-accent-400/15 px-0.5 text-accent-300">
              {match[1]}
            </mark>
          );
        }

        return <Fragment key={index}>{part}</Fragment>;
      })}
    </>
  );
}

export default Highlight;
