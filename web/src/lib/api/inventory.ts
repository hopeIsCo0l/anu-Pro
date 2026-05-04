import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";

// ── Types ─────────────────────────────────────────────────────────────────────

export interface Item {
  id: string;
  sku: string;
  name: string;
  description?: string;
  unit: string;
  category?: string;
  min_stock_qty?: string;
  attributes: Record<string, unknown>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Location {
  id: string;
  code: string;
  name: string;
  parent_id?: string;
  type: string;
  is_active: boolean;
}

export interface Lot {
  id: string;
  item_id: string;
  lot_number: string;
  supplier_ref?: string;
  received_at: string;
  expires_at?: string;
  status: string;
}

export interface StockLevel {
  lot_id: string;
  location_id: string;
  quantity: string;
  item_id: string;
  item_name: string;
  item_sku: string;
  unit: string;
  lot_number: string;
  location_code: string;
  location_name: string;
}

export interface StockMovement {
  id: number;
  lot_id: string;
  location_id: string;
  quantity: string;
  movement_type: string;
  unit_cost?: string;
  notes?: string;
  actor_id: string;
  created_at: string;
}

// ── API functions ─────────────────────────────────────────────────────────────

export const inventoryApi = {
  listItems: (params?: { q?: string; category?: string; limit?: number; offset?: number }) =>
    apiFetch<{ items: Item[] }>(`/api/items?${new URLSearchParams(
      Object.entries(params ?? {}).filter(([, v]) => v != null).map(([k, v]) => [k, String(v)])
    )}`),

  getItem: (id: string) => apiFetch<Item>(`/api/items/${id}`),

  createItem: (body: {
    sku: string; name: string; unit: string;
    description?: string; category?: string; min_stock_qty?: string;
  }) => apiFetch<Item>("/api/items", { method: "POST", body: JSON.stringify(body) }),

  updateItem: (id: string, body: Partial<Pick<Item, "name" | "description" | "unit" | "category" | "min_stock_qty">>) =>
    apiFetch<Item>(`/api/items/${id}`, { method: "PATCH", body: JSON.stringify(body) }),

  deactivateItem: (id: string) =>
    apiFetch<void>(`/api/items/${id}`, { method: "DELETE" }),

  listLocations: () => apiFetch<{ locations: Location[] }>("/api/locations"),

  createLocation: (body: { code: string; name: string; type?: string; parent_id?: string }) =>
    apiFetch<Location>("/api/locations", { method: "POST", body: JSON.stringify(body) }),

  listLots: (params?: { item_id?: string; limit?: number; offset?: number }) =>
    apiFetch<{ lots: Lot[] }>(`/api/lots?${new URLSearchParams(
      Object.entries(params ?? {}).filter(([, v]) => v != null).map(([k, v]) => [k, String(v)])
    )}`),

  createLot: (body: { item_id: string; lot_number: string; supplier_ref?: string; expires_at?: string }) =>
    apiFetch<Lot>("/api/lots", { method: "POST", body: JSON.stringify(body) }),

  getCurrentStock: (params?: { item_id?: string }) =>
    apiFetch<{ stock: StockLevel[] }>(`/api/stock/current${params?.item_id ? `?item_id=${params.item_id}` : ""}`),

  listMovements: (params?: { lot_id?: string; limit?: number }) =>
    apiFetch<{ movements: StockMovement[] }>(`/api/stock/movements?${new URLSearchParams(
      Object.entries(params ?? {}).filter(([, v]) => v != null).map(([k, v]) => [k, String(v)])
    )}`),

  recordMovement: (body: {
    lot_id: string; location_id: string; quantity: string;
    movement_type: string; idempotency_key: string; unit_cost?: string; notes?: string;
  }) => apiFetch<StockMovement>("/api/stock/movements", { method: "POST", body: JSON.stringify(body) }),
};

// ── Hooks ─────────────────────────────────────────────────────────────────────

export const useItems = (params?: Parameters<typeof inventoryApi.listItems>[0]) =>
  useQuery({
    queryKey: ["items", params],
    queryFn: () => inventoryApi.listItems(params),
    select: (d) => d.items,
  });

export const useItem = (id: string) =>
  useQuery({
    queryKey: ["items", id],
    queryFn: () => inventoryApi.getItem(id),
    enabled: !!id,
  });

export const useCreateItem = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: inventoryApi.createItem,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["items"] }),
  });
};

export const useLocations = () =>
  useQuery({
    queryKey: ["locations"],
    queryFn: inventoryApi.listLocations,
    select: (d) => d.locations,
  });

export const useCreateLocation = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: inventoryApi.createLocation,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["locations"] }),
  });
};

export const useLots = (params?: Parameters<typeof inventoryApi.listLots>[0]) =>
  useQuery({
    queryKey: ["lots", params],
    queryFn: () => inventoryApi.listLots(params),
    select: (d) => d.lots,
  });

export const useCurrentStock = (params?: { item_id?: string }) =>
  useQuery({
    queryKey: ["stock", "current", params],
    queryFn: () => inventoryApi.getCurrentStock(params),
    select: (d) => d.stock,
  });

export const useRecordMovement = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: inventoryApi.recordMovement,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["stock"] });
      qc.invalidateQueries({ queryKey: ["lots"] });
    },
  });
};
