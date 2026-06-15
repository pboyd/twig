import { memo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { Components } from "react-markdown";

export interface MarkdownProps {
  children: string;
  mode?: "block" | "inline";
  className?: string;
}

const BLOCK_COMPONENTS: Components = {
  a: ({ href, children, ...props }) => (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="text-blue-700 dark:text-blue-400 hover:underline"
      {...props}
    >
      {children}
    </a>
  ),
  pre: ({ children, ...props }) => (
    <pre className="overflow-x-auto rounded bg-gray-100 dark:bg-gray-800 p-3 text-sm" {...props}>
      {children}
    </pre>
  ),
  code: ({ children, className, ...props }) => (
    <code className={["font-mono text-sm", className].filter(Boolean).join(" ")} {...props}>
      {children}
    </code>
  ),
  table: ({ children, ...props }) => (
    <div className="overflow-x-auto">
      <table className="min-w-full text-sm border-collapse" {...props}>
        {children}
      </table>
    </div>
  ),
  th: ({ children, ...props }) => (
    <th className="border border-gray-300 dark:border-gray-600 px-3 py-1 bg-gray-50 dark:bg-gray-700 text-left font-semibold" {...props}>
      {children}
    </th>
  ),
  td: ({ children, ...props }) => (
    <td className="border border-gray-300 dark:border-gray-600 px-3 py-1" {...props}>
      {children}
    </td>
  ),
  blockquote: ({ children, ...props }) => (
    <blockquote className="border-l-4 border-gray-300 dark:border-gray-600 pl-3 text-gray-600 dark:text-gray-400 italic" {...props}>
      {children}
    </blockquote>
  ),
  h1: ({ children, ...props }) => (
    <h1 className="text-xl font-bold mt-4 mb-2 text-gray-900 dark:text-gray-100" {...props}>{children}</h1>
  ),
  h2: ({ children, ...props }) => (
    <h2 className="text-lg font-bold mt-3 mb-1 text-gray-900 dark:text-gray-100" {...props}>{children}</h2>
  ),
  h3: ({ children, ...props }) => (
    <h3 className="text-base font-semibold mt-3 mb-1 text-gray-900 dark:text-gray-100" {...props}>{children}</h3>
  ),
  h4: ({ children, ...props }) => (
    <h4 className="text-sm font-semibold mt-2 mb-1 text-gray-900 dark:text-gray-100" {...props}>{children}</h4>
  ),
  h5: ({ children, ...props }) => (
    <h5 className="text-sm font-medium mt-2 mb-1 text-gray-900 dark:text-gray-100" {...props}>{children}</h5>
  ),
  h6: ({ children, ...props }) => (
    <h6 className="text-xs font-medium mt-2 mb-1 text-gray-900 dark:text-gray-100" {...props}>{children}</h6>
  ),
  ul: ({ children, ...props }) => (
    <ul className="list-disc list-inside space-y-0.5 pl-2" {...props}>{children}</ul>
  ),
  ol: ({ children, ...props }) => (
    <ol className="list-decimal list-inside space-y-0.5 pl-2" {...props}>{children}</ol>
  ),
  hr: ({ ...props }) => (
    <hr className="border-gray-200 dark:border-gray-700 my-3" {...props} />
  ),
};

const INLINE_ALLOWED = ["strong", "em", "del", "code", "a"] as const;

function BlockMarkdown({ children, className }: { children: string; className?: string }) {
  if (!children.trim()) return null;
  return (
    <div className={["prose-sm text-sm text-gray-700 dark:text-gray-300 space-y-2", className].filter(Boolean).join(" ")}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={BLOCK_COMPONENTS}>
        {children}
      </ReactMarkdown>
    </div>
  );
}

function InlineMarkdown({ children, className }: { children: string; className?: string }) {
  if (!children.trim()) return null;
  return (
    <span className={className}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        allowedElements={[...INLINE_ALLOWED]}
        unwrapDisallowed
        components={{
          a: ({ href, children: aChildren, ...props }) => (
            <a
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-700 dark:text-blue-400 hover:underline"
              {...props}
            >
              {aChildren}
            </a>
          ),
        }}
      >
        {children}
      </ReactMarkdown>
    </span>
  );
}

export const Markdown = memo(function Markdown({ children, mode = "block", className }: MarkdownProps) {
  if (mode === "inline") {
    return <InlineMarkdown className={className}>{children}</InlineMarkdown>;
  }
  return <BlockMarkdown className={className}>{children}</BlockMarkdown>;
});
