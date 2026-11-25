import { useQuery } from "@tanstack/react-query";

import { http } from "../api/http";

export interface HealthPayload {
  service: string;
  status: string;
  timestamp: string;
  version: string;
}

export function useHealthQuery() {
  return useQuery({
    queryKey: ["health", "go-api"] as const,
    queryFn: () => http.get<HealthPayload>("/health"),
    refetchInterval: 30_000,
  });
}
