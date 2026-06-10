import { describe, it, expect, vi, beforeEach } from "vitest";
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
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/cli/download");
    expect(link).toHaveAttribute("download");
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

    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/cli/download");
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

    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/cli/download");
  });
});
