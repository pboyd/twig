import { describe, it, expect, vi } from "vitest";
import { render, screen, act, fireEvent } from "@testing-library/react";
import { ToastProvider, useToast } from "./ToastProvider";

function Trigger({ message, tone }: { message: string; tone?: "success" | "error" }) {
  const { show } = useToast();
  return <button onClick={() => show(message, tone)}>show</button>;
}

function renderWithProvider(message = "Hello!", tone?: "success" | "error") {
  return render(
    <ToastProvider>
      <Trigger message={message} tone={tone} />
    </ToastProvider>
  );
}

describe("ToastProvider", () => {
  it("renders a toast message after show() is called", () => {
    renderWithProvider("Something happened!");
    fireEvent.click(screen.getByText("show"));
    expect(screen.getByText("Something happened!")).toBeTruthy();
  });

  it("auto-dismisses after ~3 seconds", () => {
    vi.useFakeTimers();
    renderWithProvider("Fading out");
    fireEvent.click(screen.getByText("show"));
    expect(screen.getByText("Fading out")).toBeTruthy();
    act(() => { vi.advanceTimersByTime(3100); });
    expect(screen.queryByText("Fading out")).toBeNull();
    vi.useRealTimers();
  });

  it("multiple toasts stack and each auto-dismiss independently", () => {
    vi.useFakeTimers();
    function MultiTrigger() {
      const { show } = useToast();
      return (
        <>
          <button onClick={() => show("First")}>first</button>
          <button onClick={() => show("Second")}>second</button>
        </>
      );
    }
    render(<ToastProvider><MultiTrigger /></ToastProvider>);
    fireEvent.click(screen.getByText("first"));
    act(() => { vi.advanceTimersByTime(100); });
    fireEvent.click(screen.getByText("second"));
    expect(screen.getByText("First")).toBeTruthy();
    expect(screen.getByText("Second")).toBeTruthy();
    act(() => { vi.advanceTimersByTime(3000); });
    expect(screen.queryByText("First")).toBeNull();
    expect(screen.queryByText("Second")).toBeNull();
    vi.useRealTimers();
  });

  it("tap-to-dismiss removes the toast immediately", () => {
    renderWithProvider("Tap me");
    fireEvent.click(screen.getByText("show"));
    const toast = screen.getByTestId("toast");
    fireEvent.click(toast);
    expect(screen.queryByTestId("toast")).toBeNull();
  });
});
