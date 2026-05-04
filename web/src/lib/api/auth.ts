import { apiFetch } from "@/lib/api";

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user_id: string;
  tenant_id: string;
  tenant_slug: string;
  role: string;
}

export const authApi = {
  login: (email: string, password: string) =>
    apiFetch<AuthResponse>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),

  signup: (body: {
    tenant_name: string; tenant_slug: string;
    email: string; full_name: string; password: string;
  }) => apiFetch<AuthResponse>("/api/auth/signup", { method: "POST", body: JSON.stringify(body) }),

  refresh: (refresh_token: string) =>
    apiFetch<AuthResponse>("/api/auth/refresh", {
      method: "POST",
      body: JSON.stringify({ refresh_token }),
    }),

  logout: (refresh_token: string) =>
    apiFetch<void>("/api/auth/logout", {
      method: "POST",
      body: JSON.stringify({ refresh_token }),
    }),
};

export function saveTokens(res: AuthResponse) {
  localStorage.setItem("token", res.access_token);
  localStorage.setItem("refresh_token", res.refresh_token);
}

export function clearTokens() {
  localStorage.removeItem("token");
  localStorage.removeItem("refresh_token");
}

export function getToken() {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}
