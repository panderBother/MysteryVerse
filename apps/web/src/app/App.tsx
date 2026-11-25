import { ErrorBoundary } from "./ErrorBoundary";
import { AppProviders } from "./providers";
import { AppRouter } from "./router";

export function App(): React.JSX.Element {
  return (
    <AppProviders>
      <ErrorBoundary>
        <AppRouter />
      </ErrorBoundary>
    </AppProviders>
  );
}
