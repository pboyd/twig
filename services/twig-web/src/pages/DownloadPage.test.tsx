import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import DownloadPage from "./DownloadPage";
import { messages } from "../theme/messages";

vi.mock("../components/AppHeader", () => ({
  AppHeader: () => <header data-testid="app-header" />,
}));

const sampleInfo = {
  version: "v1.2.3",
  filename: "twig",
  os: "linux",
  arch: "amd64",
  label: "Linux (x86-64)",
  size: 13631488,
  sha256: "deadbeef1234",
};

function renderPage() {
  return render(
    <MemoryRouter>
      <DownloadPage />
    </MemoryRouter>,
  );
}

describe("DownloadPage", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("renders the platform label", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(sampleInfo),
      }),
    );

    renderPage();
    expect(screen.getByText(messages.downloadPlatformLabel)).toBeInTheDocument();
    // Flush the pending /cli/info fetch so its state update doesn't leak past the test.
    await screen.findByText(sampleInfo.version);
  });

  it("download link points at /cli/download", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(sampleInfo),
      }),
    );

    renderPage();
    const downloadLink = screen.getByRole("link", { name: /download/i });
    expect(downloadLink).toHaveAttribute("href", "/cli/download");
    expect(downloadLink).toHaveAttribute("download");
    // Flush the pending /cli/info fetch so its state update doesn't leak past the test.
    await screen.findByText(sampleInfo.version);
  });

  it("shows version and checksum on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(sampleInfo),
      }),
    );

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("v1.2.3")).toBeInTheDocument();
    });
    expect(screen.getByText("deadbeef1234")).toBeInTheDocument();
  });

  it("shows error state when fetch fails, download still offered", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new Error("network error")),
    );

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        messages.downloadInfoError,
      );
    });

    const downloadLink = screen.getByRole("link", { name: /download/i });
    expect(downloadLink).toHaveAttribute("href", "/cli/download");
  });

  it("shows error state when /cli/info returns 503, download still offered", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 503 }),
    );

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        messages.downloadInfoError,
      );
    });

    const downloadLink = screen.getByRole("link", { name: /download/i });
    expect(downloadLink).toHaveAttribute("href", "/cli/download");
  });

  describe("configuration section", () => {
    const ORIGIN = "http://test.example.com";

    beforeEach(() => {
      vi.stubGlobal("location", { origin: ORIGIN });
    });

    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("renders the server address matching window.location.origin (C3)", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(sampleInfo),
        }),
      );

      renderPage();
      await screen.findByText(ORIGIN);
      expect(screen.getByText(ORIGIN)).toBeInTheDocument();
    });

    it("renders a link to /account to create an API key (C4)", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(sampleInfo),
        }),
      );

      renderPage();
      await screen.findByText(sampleInfo.version);

      const accountLink = screen.getByRole("link", { name: /account/i });
      expect(accountLink).toHaveAttribute("href", "/account");
    });

    it("renders env-var command with TWIG_API_KEY and TWIG_ADDR inside a code block (C5)", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(sampleInfo),
        }),
      );

      renderPage();
      await screen.findByText(sampleInfo.version);

      const cmdEl = screen.getByText(/TWIG_API_KEY/);
      expect(cmdEl).toBeInTheDocument();
      expect(cmdEl.closest("pre")).not.toBeNull();
      expect(screen.getByText(/TWIG_ADDR/)).toBeInTheDocument();
    });

    it("renders the config-file path ~/.config/twig/config.toml (C6)", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(sampleInfo),
        }),
      );

      renderPage();
      await screen.findByText(sampleInfo.version);

      expect(
        screen.getByText(/~\/\.config\/twig\/config\.toml/),
      ).toBeInTheDocument();
    });

    it("renders the precedence note that env vars win (C7)", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(sampleInfo),
        }),
      );

      renderPage();
      await screen.findByText(sampleInfo.version);

      expect(
        screen.getByText(messages.downloadConfig.precedenceNote),
      ).toBeInTheDocument();
    });

    it("renders ./twig as a copy-paste command inside a code block (C8)", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(sampleInfo),
        }),
      );

      renderPage();
      await screen.findByText(sampleInfo.version);

      const cmdEl = screen.getByText(messages.downloadConfig.launchCommand);
      expect(cmdEl).toBeInTheDocument();
      expect(cmdEl.closest("pre")).not.toBeNull();
      expect(
        screen.getByText(messages.downloadConfig.launchHint),
      ).toBeInTheDocument();
    });
  });
});
