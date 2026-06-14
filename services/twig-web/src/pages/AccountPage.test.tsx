import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import AccountPage from "./AccountPage";
import { ToastProvider } from "../context/ToastProvider";
import { messages } from "../theme/messages";

const m = messages.account;

vi.mock("../gen/account/v1/account-AccountService_connectquery", () => ({
  changePassword: "schema:changePassword",
  listApiKeys: "schema:listApiKeys",
  createApiKey: "schema:createApiKey",
  revokeApiKey: "schema:revokeApiKey",
}));

vi.mock("../components/AppHeader", () => ({
  AppHeader: () => <header data-testid="app-header" />,
}));

const mockInvalidateQueries = vi.fn();
vi.mock("@tanstack/react-query", async (importActual) => {
  const actual = await importActual<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: vi.fn(() => ({ invalidateQueries: mockInvalidateQueries })),
  };
});

let listKeysResult: ReturnType<typeof vi.fn>;
const mockChangePassword = vi.fn();
const mockCreateApiKey = vi.fn();
const mockRevokeApiKey = vi.fn();

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: vi.fn((_schema: string) => listKeysResult()),
  useMutation: vi.fn((schema: string) => {
    if (schema === "schema:changePassword") {
      return { mutateAsync: mockChangePassword, isPending: false };
    }
    if (schema === "schema:createApiKey") {
      return { mutateAsync: mockCreateApiKey, isPending: false };
    }
    if (schema === "schema:revokeApiKey") {
      return { mutateAsync: mockRevokeApiKey, isPending: false };
    }
    return { mutateAsync: vi.fn(), isPending: false };
  }),
  createConnectQueryKey: vi.fn(() => ["mock-key"]),
}));

function makeKey(id: bigint, label: string) {
  return {
    id,
    label,
    createdAt: { seconds: 1749772800n, nanos: 0 },
    $typeName: "account.v1.ApiKeyMetadata" as const,
  };
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <ToastProvider>
          <AccountPage />
        </ToastProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mockInvalidateQueries.mockResolvedValue(undefined);
  listKeysResult = vi.fn().mockReturnValue({
    data: { keys: [] },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
});

// ---- US1: Change password ----

describe("AccountPage — password change (US1)", () => {
  it("shows error when confirmation does not match new password", async () => {
    renderPage();

    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "oldpass" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "newpassword" } });
    fireEvent.change(screen.getByLabelText("Confirm new password"), { target: { value: "doesnotmatch" } });
    fireEvent.click(screen.getByRole("button", { name: /update password/i }));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(m.passwordMismatch);
    });
    expect(mockChangePassword).not.toHaveBeenCalled();
  });

  it("shows error when new password is under 8 characters", async () => {
    renderPage();

    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "oldpass" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "short" } });
    fireEvent.change(screen.getByLabelText("Confirm new password"), { target: { value: "short" } });
    fireEvent.click(screen.getByRole("button", { name: /update password/i }));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(m.passwordTooShort);
    });
    expect(mockChangePassword).not.toHaveBeenCalled();
  });

  it("shows success toast and clears form on happy path", async () => {
    mockChangePassword.mockResolvedValue({});
    renderPage();

    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "oldpass" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "newpassword123" } });
    fireEvent.change(screen.getByLabelText("Confirm new password"), { target: { value: "newpassword123" } });
    fireEvent.click(screen.getByRole("button", { name: /update password/i }));

    await waitFor(() => {
      expect(screen.getByText(m.passwordChanged)).toBeInTheDocument();
    });
  });
});

// ---- US2: Create API key ----

describe("AccountPage — create API key (US2)", () => {
  it("shows revealed secret once with warning after key creation", async () => {
    mockCreateApiKey.mockResolvedValue({
      key: makeKey(1n, "my key"),
      secret: "abc123secretvalue",
    });
    renderPage();

    fireEvent.click(screen.getByRole("button", { name: /create new key/i }));

    await waitFor(() => {
      expect(screen.getByText("abc123secretvalue")).toBeInTheDocument();
      expect(screen.getByText(m.keySecretWarning)).toBeInTheDocument();
    });
  });

  it("secret is not visible before creation", () => {
    renderPage();
    expect(screen.queryByText(m.keySecretWarning)).not.toBeInTheDocument();
  });
});

// ---- US3: View and revoke API keys ----

describe("AccountPage — view and revoke keys (US3)", () => {
  it("shows label and creation date only (no secret)", () => {
    listKeysResult = vi.fn().mockReturnValue({
      data: { keys: [makeKey(1n, "cli tool"), makeKey(2n, "web key")] },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });

    renderPage();

    expect(screen.getByText("cli tool")).toBeInTheDocument();
    expect(screen.getByText("web key")).toBeInTheDocument();
    // No raw secrets shown.
    expect(screen.queryByText(/sha256|hex|[0-9a-f]{64}/i)).not.toBeInTheDocument();
  });

  it("shows revoke confirmation before revoking", async () => {
    listKeysResult = vi.fn().mockReturnValue({
      data: { keys: [makeKey(1n, "cli tool"), makeKey(2n, "backup")] },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });

    renderPage();

    const revokeButtons = screen.getAllByRole("button", { name: /revoke/i });
    fireEvent.click(revokeButtons[0]);

    await waitFor(() => {
      expect(screen.getByText(m.revokeConfirm)).toBeInTheDocument();
    });
    expect(mockRevokeApiKey).not.toHaveBeenCalled();
  });

  it("shows last-key warning when only one key remains", async () => {
    listKeysResult = vi.fn().mockReturnValue({
      data: { keys: [makeKey(1n, "only key")] },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });

    renderPage();

    fireEvent.click(screen.getByRole("button", { name: /revoke/i }));

    await waitFor(() => {
      expect(screen.getByText(m.revokeLastKeyWarning)).toBeInTheDocument();
    });
  });

  it("revokes after confirmation and shows toast", async () => {
    mockRevokeApiKey.mockResolvedValue({});
    listKeysResult = vi.fn().mockReturnValue({
      data: { keys: [makeKey(1n, "cli tool"), makeKey(2n, "backup")] },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });

    renderPage();

    const revokeButtons = screen.getAllByRole("button", { name: /revoke/i });
    fireEvent.click(revokeButtons[0]);

    // Confirm dialog appears; click the red "Revoke" confirm button (first among confirm buttons).
    await waitFor(() => screen.getByText(m.revokeConfirm));
    const confirmButtons = screen.getAllByRole("button", { name: /^revoke$/i });
    // The red confirmation button is the one we want; it appears in the pending row.
    fireEvent.click(confirmButtons[0]);

    await waitFor(() => {
      expect(mockRevokeApiKey).toHaveBeenCalledWith({ id: 1n });
    });
    await waitFor(() => {
      expect(screen.getByText(m.keyRevoked)).toBeInTheDocument();
    });
  });
});
