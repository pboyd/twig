import { useState, type FormEvent } from "react";
import { useQuery, useMutation, createConnectQueryKey } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { ConnectError, Code } from "@connectrpc/connect";
import {
  changePassword,
  listApiKeys,
  createApiKey,
  revokeApiKey,
} from "../gen/account/v1/account-AccountService_connectquery";
import type { ApiKeyMetadata } from "../gen/account/v1/account_pb";
import { AppHeader } from "../components/AppHeader";
import { Field } from "../components/Field";
import { Button } from "../components/Button";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { useToast } from "../context/ToastProvider";
import { messages } from "../theme/messages";

const m = messages.account;

function formatDate(ts: { seconds: bigint } | undefined): string {
  if (!ts) return "";
  return new Date(Number(ts.seconds) * 1000).toLocaleDateString();
}

export default function AccountPage() {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <AppHeader />
      <main className="mx-auto max-w-2xl px-4 py-10 space-y-10">
        <PasswordSection />
        <ApiKeysSection />
      </main>
    </div>
  );
}

function PasswordSection() {
  const { show } = useToast();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const { mutateAsync: doChangePassword, isPending } = useMutation(changePassword);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);

    if (newPassword !== confirmPassword) {
      setError(m.passwordMismatch);
      return;
    }
    if (newPassword.length < 8) {
      setError(m.passwordTooShort);
      return;
    }

    try {
      await doChangePassword({ currentPassword, newPassword });
      show(m.passwordChanged);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (err) {
      if (err instanceof ConnectError) {
        if (err.code === Code.ResourceExhausted) {
          setError(m.passwordRateLimited);
        } else if (err.code === Code.FailedPrecondition) {
          setError(m.passwordWrongCurrent);
        } else {
          setError(m.passwordChangeFailed);
        }
      } else {
        setError(m.passwordChangeFailed);
      }
    }
  }

  return (
    <section aria-labelledby="password-heading">
      <h2
        id="password-heading"
        className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4"
      >
        {m.passwordHeading}
      </h2>
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6">
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <Field
            id="current-password"
            label={m.currentPasswordLabel}
            type="password"
            autoComplete="current-password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            required
          />
          <Field
            id="new-password"
            label={m.newPasswordLabel}
            type="password"
            autoComplete="new-password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            required
          />
          <Field
            id="confirm-password"
            label={m.confirmPasswordLabel}
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            required
          />
          {error && <ErrorBanner message={error} />}
          <Button type="submit" loading={isPending} className="self-start">
            {m.changePasswordButton}
          </Button>
        </form>
      </div>
    </section>
  );
}

function ApiKeysSection() {
  const { show } = useToast();
  const queryClient = useQueryClient();
  const [newLabel, setNewLabel] = useState("");
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [revokingId, setRevokingId] = useState<bigint | null>(null);
  const [pendingRevokeId, setPendingRevokeId] = useState<bigint | null>(null);

  const keysQueryKey = createConnectQueryKey({ schema: listApiKeys, input: {}, cardinality: "finite" });

  const keysQuery = useQuery(listApiKeys, {});
  const { mutateAsync: doCreate, isPending: isCreating } = useMutation(createApiKey);
  const { mutateAsync: doRevoke } = useMutation(revokeApiKey);

  const keys: ApiKeyMetadata[] = keysQuery.data?.keys ?? [];

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    try {
      const res = await doCreate({ label: newLabel });
      setRevealedSecret(res.secret);
      setCopied(false);
      setNewLabel("");
      await queryClient.invalidateQueries({ queryKey: keysQueryKey });
    } catch {
      show(messages.connectivityError, "error");
    }
  }

  async function handleCopy() {
    if (!revealedSecret) return;
    await navigator.clipboard.writeText(revealedSecret);
    setCopied(true);
    show(m.keyCopied);
  }

  function confirmRevoke(id: bigint) {
    setPendingRevokeId(id);
  }

  async function executeRevoke(id: bigint) {
    setPendingRevokeId(null);
    setRevokingId(id);
    try {
      await doRevoke({ id });
      show(m.keyRevoked);
      await queryClient.invalidateQueries({ queryKey: keysQueryKey });
    } catch {
      show(m.revokeFailed, "error");
    } finally {
      setRevokingId(null);
    }
  }

  return (
    <section aria-labelledby="apikeys-heading">
      <h2
        id="apikeys-heading"
        className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4"
      >
        {m.apiKeysHeading}
      </h2>
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 space-y-6">
        {/* Create new key */}
        <form onSubmit={handleCreate} className="flex gap-3 items-end">
          <div className="flex-1">
            <Field
              id="key-label"
              label="Label"
              type="text"
              placeholder={m.keyLabelPlaceholder}
              value={newLabel}
              onChange={(e) => setNewLabel(e.target.value)}
            />
          </div>
          <Button type="submit" loading={isCreating} className="shrink-0">
            {m.createKeyButton}
          </Button>
        </form>

        {/* Revealed secret */}
        {revealedSecret && (
          <div
            role="status"
            className="rounded-md bg-amber-50 dark:bg-amber-900/30 border border-amber-200 dark:border-amber-700 p-4 space-y-3"
          >
            <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
              {m.keySecretWarning}
            </p>
            <div className="flex items-center gap-3">
              <code className="flex-1 text-xs font-mono break-all text-amber-900 dark:text-amber-100">
                {revealedSecret}
              </code>
              <Button variant="secondary" onClick={handleCopy} className="shrink-0 text-xs px-3">
                {copied ? m.keyCopied : m.copyKeyButton}
              </Button>
            </div>
          </div>
        )}

        {/* Keys list */}
        {keysQuery.isLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <Spinner size="sm" />
            <span>Loading…</span>
          </div>
        )}
        {keysQuery.isError && (
          <ErrorBanner message={m.keysLoadError} onRetry={() => keysQuery.refetch()} />
        )}
        {!keysQuery.isLoading && !keysQuery.isError && keys.length === 0 && (
          <p className="text-sm text-gray-500 dark:text-gray-400">{m.noKeys}</p>
        )}
        {keys.length > 0 && (
          <ul className="divide-y divide-gray-100 dark:divide-gray-700">
            {keys.map((key) => {
              const isLast = keys.length === 1;
              const isPendingRevoke = pendingRevokeId === key.id;
              return (
                <li key={String(key.id)} className="flex items-center justify-between py-3 gap-4">
                  <div>
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {key.label || "(unlabelled)"}
                    </span>
                    <span className="block text-xs text-gray-500 dark:text-gray-400">
                      {m.createdAt(formatDate(key.createdAt))}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    {isPendingRevoke ? (
                      <>
                        <span className="text-xs text-gray-600 dark:text-gray-400">
                          {isLast ? m.revokeLastKeyWarning : m.revokeConfirm}
                        </span>
                        <Button
                          variant="secondary"
                          onClick={() => setPendingRevokeId(null)}
                          className="text-xs px-2"
                        >
                          Cancel
                        </Button>
                        <Button
                          onClick={() => executeRevoke(key.id)}
                          className="text-xs px-2 bg-red-600 hover:bg-red-700 text-white border-red-600"
                        >
                          Revoke
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="secondary"
                        onClick={() => confirmRevoke(key.id)}
                        loading={revokingId === key.id}
                        className="text-xs px-3"
                      >
                        {m.revokeKeyButton}
                      </Button>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}
