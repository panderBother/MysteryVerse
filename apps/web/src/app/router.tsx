import { createBrowserRouter, RouterProvider } from "react-router-dom";

import { FoundationPage } from "../pages/foundation/FoundationPage";
import { NotFoundPage } from "../pages/not-found/NotFoundPage";

const router = createBrowserRouter([
  {
    path: "/",
    element: <FoundationPage />,
  },
  {
    path: "*",
    element: <NotFoundPage />,
  },
]);

export function AppRouter(): React.JSX.Element {
  return <RouterProvider router={router} />;
}
