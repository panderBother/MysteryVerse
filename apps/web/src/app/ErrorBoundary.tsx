import { Component, type ErrorInfo, type ReactNode } from "react";

import { toAppError } from "../lib/errors";

interface ErrorBoundaryProps {
  children: ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  public override state: ErrorBoundaryState = {
    error: null,
  };

  public static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  public override componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error("Unhandled application error", toAppError(error), info.componentStack);
  }

  public override render(): ReactNode {
    if (this.state.error) {
      return (
        <main className="grid min-h-screen place-items-center bg-canvas px-6 text-content-primary">
          <section className="w-full max-w-xl rounded-card border border-border-subtle bg-surface p-6">
            <p className="font-mono text-caption text-state-danger">APPLICATION_ERROR</p>
            <h1 className="mt-3 font-ui text-h2">应用暂时无法继续运行</h1>
            <p className="mt-3 text-body text-content-secondary">
              请刷新页面；如果问题持续存在，请保留浏览器控制台中的错误信息。
            </p>
            <button
              className="mt-6 min-h-11 rounded-control bg-action px-4 text-ui font-semibold text-action-ink"
              onClick={() => window.location.reload()}
              type="button"
            >
              刷新页面
            </button>
          </section>
        </main>
      );
    }

    return this.props.children;
  }
}
