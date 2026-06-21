/** Operational endpoints (`/health` and `/ready`) — no auth required. */

import { api } from "./client";

export interface HealthStatus {
  status: string;
}

export function getHealth(signal?: AbortSignal): Promise<HealthStatus> {
  return api.get<HealthStatus>("/health", { signal });
}

export function getReady(signal?: AbortSignal): Promise<HealthStatus> {
  return api.get<HealthStatus>("/ready", { signal });
}
