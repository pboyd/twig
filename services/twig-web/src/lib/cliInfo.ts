export interface CLIInfo {
  version: string;
  filename: string;
  os: string;
  arch: string;
  label: string;
  size: number;
  sha256: string;
}

export async function fetchCliInfo(): Promise<CLIInfo> {
  const res = await fetch("/cli/info", { credentials: "include" });
  if (!res.ok) {
    throw new Error(`/cli/info returned ${res.status}`);
  }
  return res.json() as Promise<CLIInfo>;
}
