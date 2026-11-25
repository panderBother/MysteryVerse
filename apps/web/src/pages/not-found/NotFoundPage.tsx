import { Link } from "react-router-dom";

export function NotFoundPage(): React.JSX.Element {
  return (
    <main className="grid min-h-screen place-items-center bg-canvas px-6 text-content-primary">
      <section className="w-full max-w-lg rounded-card border border-border-subtle bg-surface p-8">
        <p className="font-mono text-caption text-state-warning">404 / NOT_FOUND</p>
        <h1 className="mt-3 font-narrative text-h2">当前页面不存在</h1>
        <Link
          className="mt-6 inline-flex min-h-11 items-center rounded-control border border-border-strong px-4 text-ui font-semibold"
          to="/"
        >
          返回应用入口
        </Link>
      </section>
    </main>
  );
}
