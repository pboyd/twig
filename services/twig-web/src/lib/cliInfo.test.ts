import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchCliInfo } from "./cliInfo";

const sampleInfo = {
  version: "v1.0.0",
  filename: "twig",
  os: "linux",
  arch: "amd64",
  label: "Linux (x86-64)",
  size: 12345678,
  sha256: "abc123",
};

describe("fetchCliInfo", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("returns parsed metadata on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(sampleInfo),
      }),
    );

    const result = await fetchCliInfo();
    expect(result).toEqual(sampleInfo);
    expect(fetch).toHaveBeenCalledWith("/cli/info", { credentials: "include" });
  });

  it("throws on non-200 response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 503 }),
    );

    await expect(fetchCliInfo()).rejects.toThrow("/cli/info returned 503");
  });

  it("throws on network error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new Error("network failure")),
    );

    await expect(fetchCliInfo()).rejects.toThrow("network failure");
  });
});
