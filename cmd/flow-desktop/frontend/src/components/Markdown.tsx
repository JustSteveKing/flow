import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

// Tailwind-styled renderers so markdown bodies read well in the dark theme
// without pulling in a typography plugin.
const components: Components = {
  h1: ({ ...p }) => <h1 className="mb-2 mt-4 text-lg font-semibold first:mt-0" {...p} />,
  h2: ({ ...p }) => <h2 className="mb-2 mt-4 text-base font-semibold first:mt-0" {...p} />,
  h3: ({ ...p }) => <h3 className="mb-1 mt-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground" {...p} />,
  p: ({ ...p }) => <p className="my-2 leading-relaxed" {...p} />,
  ul: ({ ...p }) => <ul className="my-2 ml-5 list-disc space-y-1" {...p} />,
  ol: ({ ...p }) => <ol className="my-2 ml-5 list-decimal space-y-1" {...p} />,
  li: ({ ...p }) => <li className="leading-relaxed" {...p} />,
  a: ({ ...p }) => <a className="text-primary underline underline-offset-2" {...p} />,
  blockquote: ({ ...p }) => (
    <blockquote className="my-2 border-l-2 border-border pl-3 text-muted-foreground" {...p} />
  ),
  code: ({ className, ...p }) => (
    <code className={`rounded bg-background px-1.5 py-0.5 font-mono text-[12px] text-primary ${className ?? ""}`} {...p} />
  ),
  pre: ({ ...p }) => <pre className="my-2 overflow-x-auto rounded-md bg-background p-3 text-[12px]" {...p} />,
  table: ({ ...p }) => <table className="my-2 w-full border-collapse text-sm" {...p} />,
  th: ({ ...p }) => <th className="border border-border px-2 py-1 text-left font-semibold" {...p} />,
  td: ({ ...p }) => <td className="border border-border px-2 py-1" {...p} />,
  hr: () => <hr className="my-3 border-border" />,
};

export function Markdown({ children }: { children: string }) {
  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
      {children}
    </ReactMarkdown>
  );
}
