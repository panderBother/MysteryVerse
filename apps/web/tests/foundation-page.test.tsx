import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MotionConfig } from "motion/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FoundationPage } from "../src/pages/foundation/FoundationPage";

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MotionConfig reducedMotion="always">
        <MemoryRouter>
          <FoundationPage />
        </MemoryRouter>
      </MotionConfig>
    </QueryClientProvider>,
  );
}

describe("FoundationPage", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("offline")));
  });

  it("renders the application shell", () => {
    renderPage();

    expect(screen.getByRole("heading", { name: "一个可扩展的推理游戏工程基础" })).toBeVisible();
    expect(screen.getByText("FOUNDATION / PHASE 0")).toBeVisible();
  });

  it("shows a safe unavailable state when the API cannot be reached", async () => {
    renderPage();

    expect(await screen.findByText("服务不可用")).toBeVisible();
  });
});
