import { useEffect, useState } from "react";
import { AppHeader } from "../components/AppHeader";
import { Button } from "../components/Button";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { messages } from "../theme/messages";
import { fetchCliInfo, type CLIInfo } from "../lib/cliInfo";

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function DownloadPage() {
  const [info, setInfo] = useState<CLIInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  function loadInfo() {
    setLoading(true);
    setError(false);
    fetchCliInfo()
      .then((data) => {
        setInfo(data);
        setLoading(false);
      })
      .catch(() => {
        setError(true);
        setLoading(false);
      });
  }

  useEffect(() => {
    loadInfo();
  }, []);

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <AppHeader />
      <main className="mx-auto max-w-2xl px-4 py-10">
        <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
          {messages.downloadHeading}
        </h1>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
          {messages.downloadHelperText}
        </p>

        <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {messages.downloadPlatformLabel}
            </span>
            <a href="/cli/download" download>
              <Button>{messages.downloadButtonLabel}</Button>
            </a>
          </div>

          {loading && (
            <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <Spinner size="sm" />
              <span>{messages.downloadInfoLoading}</span>
            </div>
          )}

          {error && (
            <ErrorBanner message={messages.downloadInfoError} onRetry={loadInfo} />
          )}

          {info && !loading && !error && (
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
              <dt className="text-gray-500 dark:text-gray-400">Version</dt>
              <dd className="font-mono text-gray-900 dark:text-gray-100">{info.version}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Size</dt>
              <dd className="text-gray-900 dark:text-gray-100">{formatBytes(info.size)}</dd>
              <dt className="text-gray-500 dark:text-gray-400">SHA-256</dt>
              <dd className="font-mono text-xs break-all text-gray-900 dark:text-gray-100">{info.sha256}</dd>
            </dl>
          )}
        </div>

        <p className="mt-4 text-xs text-gray-500 dark:text-gray-400">
          {messages.downloadPostHint}
        </p>
      </main>
    </div>
  );
}
