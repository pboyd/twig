import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { Markdown } from "./Markdown";

describe("Markdown — block mode", () => {
  it("renders headings", () => {
    const { container } = render(<Markdown>{"# Heading 1\n## Heading 2"}</Markdown>);
    expect(container.querySelector("h1")).toBeTruthy();
    expect(container.querySelector("h2")).toBeTruthy();
  });

  it("renders unordered list items", () => {
    const { container } = render(<Markdown>{"- alpha\n- beta"}</Markdown>);
    const items = container.querySelectorAll("li");
    expect(items.length).toBeGreaterThanOrEqual(2);
  });

  it("renders ordered list items", () => {
    const { container } = render(<Markdown>{"1. first\n2. second"}</Markdown>);
    const items = container.querySelectorAll("li");
    expect(items.length).toBeGreaterThanOrEqual(2);
  });

  it("renders GFM table", () => {
    const { container } = render(
      <Markdown>{"| a | b |\n|---|---|\n| 1 | 2 |"}</Markdown>
    );
    expect(container.querySelector("table")).toBeTruthy();
  });

  it("renders link with href", () => {
    const { container } = render(<Markdown>{"[Visit](https://example.com)"}</Markdown>);
    const a = container.querySelector("a");
    expect(a).toBeTruthy();
    expect(a?.getAttribute("href")).toBe("https://example.com");
  });

  it("renders fenced code block", () => {
    const { container } = render(<Markdown>{"```js\nconst x = 1;\n```"}</Markdown>);
    expect(container.querySelector("pre")).toBeTruthy();
    expect(container.querySelector("code")).toBeTruthy();
  });

  it("renders strikethrough", () => {
    const { container } = render(<Markdown>{"~~struck~~"}</Markdown>);
    expect(container.querySelector("del")).toBeTruthy();
  });

  it("renders task-list checkboxes", () => {
    const { container } = render(
      <Markdown>{"- [x] done\n- [ ] todo"}</Markdown>
    );
    const checkboxes = container.querySelectorAll("input[type='checkbox']");
    expect(checkboxes.length).toBeGreaterThanOrEqual(2);
  });

  it("renders blockquote", () => {
    const { container } = render(<Markdown>{"> a quote"}</Markdown>);
    expect(container.querySelector("blockquote")).toBeTruthy();
  });

  it("renders horizontal rule", () => {
    const { container } = render(<Markdown>{"---"}</Markdown>);
    expect(container.querySelector("hr")).toBeTruthy();
  });

  it("renders nothing for empty string", () => {
    const { container } = render(<Markdown>{""}</Markdown>);
    // Should render nothing meaningful — no block content
    expect(container.textContent?.trim()).toBe("");
  });

  it("renders nothing for whitespace-only input", () => {
    const { container } = render(<Markdown>{"   \n  "}</Markdown>);
    expect(container.textContent?.trim()).toBe("");
  });

  it("does not throw for malformed markdown", () => {
    expect(() => render(<Markdown>{"[unclosed link"}</Markdown>)).not.toThrow();
  });

  it("neutralizes javascript: href", () => {
    const { container } = render(
      <Markdown>{"[click](javascript:alert(1))"}</Markdown>
    );
    const a = container.querySelector("a");
    // href should be absent, empty, or not start with javascript:
    const href = a?.getAttribute("href") ?? "";
    expect(href).not.toMatch(/^javascript:/i);
  });

  it("does not produce executable script element from raw <script>", () => {
    const { container } = render(
      <Markdown>{"<script>alert(1)</script>"}</Markdown>
    );
    expect(container.querySelector("script")).toBeNull();
  });

  it("does not produce onerror attribute from <img onerror>", () => {
    const { container } = render(
      <Markdown>{"<img src=x onerror=alert(1)>"}</Markdown>
    );
    const img = container.querySelector("img");
    // Either no img is rendered, or it has no onerror attribute
    if (img) {
      expect(img.getAttribute("onerror")).toBeNull();
    } else {
      expect(img).toBeNull();
    }
  });
});

describe("Markdown — inline mode", () => {
  it("renders emphasis without block elements", () => {
    const { container } = render(
      <Markdown mode="inline">{"**bold** and _italic_"}</Markdown>
    );
    expect(container.querySelector("strong")).toBeTruthy();
    expect(container.querySelector("em")).toBeTruthy();
    // No block elements
    expect(container.querySelector("p")).toBeNull();
    expect(container.querySelector("div > p")).toBeNull();
  });

  it("degrades block syntax (# heading) to inline text", () => {
    const { container } = render(
      <Markdown mode="inline">{"# Not a heading"}</Markdown>
    );
    expect(container.querySelector("h1")).toBeNull();
    // Text should still appear
    expect(container.textContent).toContain("Not a heading");
  });

  it("degrades list marker to inline text", () => {
    const { container } = render(
      <Markdown mode="inline">{"- list item"}</Markdown>
    );
    expect(container.querySelector("ul")).toBeNull();
    expect(container.querySelector("li")).toBeNull();
  });

  it("renders del (strikethrough) inline", () => {
    const { container } = render(
      <Markdown mode="inline">{"~~struck~~"}</Markdown>
    );
    expect(container.querySelector("del")).toBeTruthy();
  });

  it("renders inline code", () => {
    const { container } = render(
      <Markdown mode="inline">{"`code`"}</Markdown>
    );
    expect(container.querySelector("code")).toBeTruthy();
  });
});
