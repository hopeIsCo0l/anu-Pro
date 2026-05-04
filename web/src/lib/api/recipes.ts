import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";

// ── Types ─────────────────────────────────────────────────────────────────────

export interface RecipeInput {
  id: string;
  item_id: string;
  item_name?: string;
  item_sku?: string;
  quantity: string;
  unit: string;
  is_optional: boolean;
  sort_order: number;
}

export interface RecipeOutput {
  id: string;
  item_id: string;
  item_name?: string;
  quantity: string;
  unit: string;
  output_type: string;
}

export interface RecipeVersion {
  id: string;
  recipe_id: string;
  version: number;
  output_qty: string;
  output_unit: string;
  yield_pct: string;
  notes?: string;
  created_at: string;
  inputs?: RecipeInput[];
  outputs?: RecipeOutput[];
}

export interface Recipe {
  id: string;
  name: string;
  output_item_id: string;
  output_item_name?: string;
  notes?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  latest_version?: RecipeVersion;
}

// ── API functions ─────────────────────────────────────────────────────────────

export const recipesApi = {
  list: (active?: boolean) =>
    apiFetch<{ recipes: Recipe[] }>(`/api/recipes${active === false ? "?active=false" : ""}`),

  get: (id: string) => apiFetch<Recipe>(`/api/recipes/${id}`),

  create: (body: {
    name: string;
    output_item_id: string;
    notes?: string;
    version: {
      output_qty: string;
      output_unit: string;
      yield_pct?: string;
      notes?: string;
      inputs: Array<{ item_id: string; quantity: string; unit: string; is_optional?: boolean; sort_order?: number }>;
      outputs?: Array<{ item_id: string; quantity: string; unit: string; output_type?: string }>;
    };
  }) => apiFetch<Recipe>("/api/recipes", { method: "POST", body: JSON.stringify(body) }),

  addVersion: (
    recipeId: string,
    body: { output_qty: string; output_unit: string; yield_pct?: string; inputs: RecipeInput[] }
  ) => apiFetch<RecipeVersion>(`/api/recipes/${recipeId}/versions`, { method: "POST", body: JSON.stringify(body) }),

  getVersion: (recipeId: string, versionId: string) =>
    apiFetch<RecipeVersion>(`/api/recipes/${recipeId}/versions/${versionId}`),
};

// ── Hooks ─────────────────────────────────────────────────────────────────────

export const useRecipes = (active?: boolean) =>
  useQuery({
    queryKey: ["recipes", { active }],
    queryFn: () => recipesApi.list(active),
    select: (d) => d.recipes,
  });

export const useRecipe = (id: string) =>
  useQuery({
    queryKey: ["recipes", id],
    queryFn: () => recipesApi.get(id),
    enabled: !!id,
  });

export const useCreateRecipe = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: recipesApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["recipes"] }),
  });
};
