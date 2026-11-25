import { MoonStar, Sun } from "lucide-react";
import { useEffect } from "react";

import { ServiceStatus } from "../../components/ServiceStatus";
import { useAppStore } from "../../stores/app-store";

export function FoundationPage(): React.JSX.Element {
  const theme = useAppStore((state) => state.theme);
  const setTheme = useAppStore((state) => state.setTheme);

  useEffect(() => {
    document.documentElement.dataset["theme"] = theme;
  }, [theme]);

  const nextTheme = theme === "archive" ? "paper" : "archive";

  return (
    <div className="min-h-screen bg-canvas text-content-primary">
      <header className="border-b border-border-subtle bg-surface">
        <div className="mx-auto flex min-h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-3">
            <span className="grid size-9 place-items-center rounded-control border border-border-strong font-mono text-ui">
              MV
            </span>
            <div>
              <p className="font-narrative text-h3">
                {import.meta.env.VITE_APP_NAME ?? "MysteryVerse"}
              </p>
              <p className="font-mono text-micro text-content-muted">FOUNDATION / PHASE 0</p>
            </div>
          </div>

          <button
            aria-label={nextTheme === "paper" ? "切换到明亮主题" : "切换到档案主题"}
            className="grid size-11 place-items-center rounded-control border border-border-subtle text-content-secondary hover:border-border-strong hover:text-content-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            onClick={() => setTheme(nextTheme)}
            type="button"
          >
            {theme === "archive" ? (
              <Sun aria-hidden="true" className="size-5" />
            ) : (
              <MoonStar aria-hidden="true" className="size-5" />
            )}
          </button>
        </div>
      </header>

      <main className="mx-auto grid max-w-6xl gap-6 px-4 py-8 sm:px-6 lg:grid-cols-[minmax(0,1fr)_320px]">
        <section className="min-h-[420px] rounded-card border border-border-subtle bg-surface p-6 sm:p-8">
          <p className="font-mono text-caption text-content-muted">WORKSPACE</p>
          <h1 className="mt-3 max-w-2xl font-narrative text-h1">一个可扩展的推理游戏工程基础</h1>
          <div className="mt-8 h-px bg-border-subtle" />
          <div className="mt-8 grid gap-4 sm:grid-cols-3">
            {["Web", "Go API", "Python AI"].map((service) => (
              <div className="border-l-2 border-border-strong pl-4" key={service}>
                <p className="font-ui text-ui font-semibold">{service}</p>
                <p className="mt-1 font-mono text-micro text-content-muted">INITIALIZED</p>
              </div>
            ))}
          </div>
        </section>

        <aside className="rounded-card border border-border-subtle bg-surface p-5">
          <p className="font-mono text-caption text-content-muted">SERVICE STATUS</p>
          <div className="mt-5">
            <ServiceStatus />
          </div>
          <dl className="mt-6 grid gap-3 text-ui">
            <div className="flex items-center justify-between gap-4">
              <dt className="text-content-muted">API</dt>
              <dd className="font-mono">/api/v1</dd>
            </div>
            <div className="flex items-center justify-between gap-4">
              <dt className="text-content-muted">Theme</dt>
              <dd className="font-mono">{theme}</dd>
            </div>
          </dl>
        </aside>
      </main>
    </div>
  );
}
