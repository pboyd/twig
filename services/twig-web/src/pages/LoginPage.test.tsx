import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import LoginPage from "./LoginPage";
import { messages } from "../theme/messages";

// react-router navigation is mocked so we can assert on it.
const mockNavigate = vi.fn();
vi.mock("react-router", async (importActual) => {
  const actual = await importActual<typeof import("react-router")>();
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

function renderPage(search = "") {
  return render(
    <MemoryRouter initialEntries={[`/login${search}`]}>
      <LoginPage />
    </MemoryRouter>,
  );
}

async function submitForm(username = "alice", password = "secret") {
  fireEvent.change(screen.getByLabelText(/username/i), {
    target: { value: username },
  });
  fireEvent.change(screen.getByLabelText(/password/i), {
    target: { value: password },
  });
  fireEvent.click(screen.getByRole("button", { name: /sign in/i }));
}

describe("LoginPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("navigates to /tasks on successful login", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: true, status: 200 }),
    );

    renderPage();
    await submitForm();

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/tasks", { replace: true });
    });
  });

  it("navigates to the ?next param on successful login", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: true, status: 200 }),
    );

    renderPage("?next=/some-page");
    await submitForm();

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/some-page", { replace: true });
    });
  });

  it("shows login failure message on 401", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 401 }),
    );

    renderPage();
    await submitForm();

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(messages.loginFailure);
    });
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("shows throttled message on 429", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 429 }),
    );

    renderPage();
    await submitForm();

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        messages.loginThrottled,
      );
    });
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("shows connectivity error on other server errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 500 }),
    );

    renderPage();
    await submitForm();

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        messages.connectivityError,
      );
    });
  });

  it("shows connectivity error on network failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new Error("network error")),
    );

    renderPage();
    await submitForm();

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        messages.connectivityError,
      );
    });
  });
});
