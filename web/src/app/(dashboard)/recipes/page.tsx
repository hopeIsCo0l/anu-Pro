"use client";

import { useState } from "react";
import { useRecipes, useCreateRecipe, type Recipe } from "@/lib/api/recipes";
import { useItems as useInventoryItems } from "@/lib/api/inventory";

export default function RecipesPage() {
  const [showCreate, setShowCreate] = useState(false);
  const { data: recipes = [], isLoading } = useRecipes();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-black">Recipes</h2>
          <p className="text-sm text-zinc-500">Bills of materials and versions</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="bg-black text-white text-sm font-medium px-4 py-2 rounded-md hover:bg-zinc-800 transition-colors"
        >
          New Recipe
        </button>
      </div>

      {isLoading ? (
        <p className="text-sm text-zinc-400">Loading…</p>
      ) : recipes.length === 0 ? (
        <div className="bg-white border border-zinc-200 rounded-lg p-8 text-center">
          <p className="text-sm text-zinc-400">No recipes yet.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {recipes.map((r) => <RecipeCard key={r.id} recipe={r} />)}
        </div>
      )}

      {showCreate && <CreateRecipeDialog onClose={() => setShowCreate(false)} />}
    </div>
  );
}

function RecipeCard({ recipe }: { recipe: Recipe }) {
  const v = recipe.latest_version;
  return (
    <div className="bg-white border border-zinc-200 rounded-lg p-5 hover:border-zinc-400 transition-colors">
      <div className="flex items-start justify-between mb-3">
        <p className="font-semibold text-black text-sm">{recipe.name}</p>
        <span className={`text-xs px-2 py-0.5 rounded font-medium ${recipe.is_active ? "bg-black text-white" : "bg-zinc-100 text-zinc-500"}`}>
          {recipe.is_active ? "Active" : "Inactive"}
        </span>
      </div>
      <p className="text-xs text-zinc-500 mb-3">Produces: <span className="text-zinc-700">{recipe.output_item_name}</span></p>
      {v && (
        <div className="text-xs text-zinc-500 space-y-1 border-t border-zinc-100 pt-3">
          <div className="flex justify-between">
            <span>Version</span><span className="text-zinc-700">v{v.version}</span>
          </div>
          <div className="flex justify-between">
            <span>Output</span><span className="text-zinc-700">{v.output_qty} {v.output_unit}</span>
          </div>
          <div className="flex justify-between">
            <span>Yield</span><span className="text-zinc-700">{v.yield_pct}%</span>
          </div>
          <div className="flex justify-between">
            <span>Inputs</span><span className="text-zinc-700">{v.inputs?.length ?? 0} ingredients</span>
          </div>
        </div>
      )}
    </div>
  );
}

function CreateRecipeDialog({ onClose }: { onClose: () => void }) {
  const createRecipe = useCreateRecipe();
  const { data: items = [] } = useInventoryItems();
  const [name, setName] = useState("");
  const [outputItemId, setOutputItemId] = useState("");
  const [outputQty, setOutputQty] = useState("");
  const [outputUnit, setOutputUnit] = useState("");
  const [yieldPct, setYieldPct] = useState("100");
  const [notes, setNotes] = useState("");
  const [inputs, setInputs] = useState<Array<{ item_id: string; quantity: string; unit: string }>>([
    { item_id: "", quantity: "", unit: "" },
  ]);
  const [error, setError] = useState("");

  function addInput() {
    setInputs((prev) => [...prev, { item_id: "", quantity: "", unit: "" }]);
  }

  function removeInput(i: number) {
    setInputs((prev) => prev.filter((_, idx) => idx !== i));
  }

  function setInput(i: number, field: keyof (typeof inputs)[0], value: string) {
    setInputs((prev) => prev.map((inp, idx) => idx === i ? { ...inp, [field]: value } : inp));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await createRecipe.mutateAsync({
        name,
        output_item_id: outputItemId,
        notes: notes || undefined,
        version: {
          output_qty: outputQty,
          output_unit: outputUnit,
          yield_pct: yieldPct || undefined,
          inputs: inputs.filter((i) => i.item_id && i.quantity && i.unit).map((i, idx) => ({
            item_id: i.item_id, quantity: i.quantity, unit: i.unit, sort_order: idx,
          })),
        },
      });
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to create recipe");
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 overflow-y-auto py-8" onClick={onClose}>
      <div className="bg-white rounded-lg w-full max-w-lg p-6 shadow-xl my-auto" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-base font-semibold text-black mb-5">New Recipe</h3>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-zinc-700 mb-1">Recipe Name *</label>
            <input required value={name} onChange={(e) => setName(e.target.value)}
              className="w-full border border-zinc-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-black" />
          </div>

          <div>
            <label className="block text-xs font-medium text-zinc-700 mb-1">Output Item *</label>
            <select required value={outputItemId} onChange={(e) => setOutputItemId(e.target.value)}
              className="w-full border border-zinc-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-black bg-white">
              <option value="">Select item…</option>
              {items.map((item) => <option key={item.id} value={item.id}>{item.name} ({item.sku})</option>)}
            </select>
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div className="col-span-2">
              <label className="block text-xs font-medium text-zinc-700 mb-1">Output Qty *</label>
              <input required type="text" value={outputQty} onChange={(e) => setOutputQty(e.target.value)} placeholder="100"
                className="w-full border border-zinc-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-black" />
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-700 mb-1">Unit *</label>
              <input required value={outputUnit} onChange={(e) => setOutputUnit(e.target.value)} placeholder="kg"
                className="w-full border border-zinc-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-black" />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-zinc-700 mb-1">Yield % (default 100)</label>
            <input type="text" value={yieldPct} onChange={(e) => setYieldPct(e.target.value)}
              className="w-full border border-zinc-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-black" />
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-xs font-medium text-zinc-700">Inputs (Ingredients)</label>
              <button type="button" onClick={addInput} className="text-xs text-zinc-500 hover:text-black">+ Add</button>
            </div>
            <div className="space-y-2">
              {inputs.map((inp, i) => (
                <div key={i} className="flex gap-2 items-start">
                  <select value={inp.item_id} onChange={(e) => setInput(i, "item_id", e.target.value)}
                    className="flex-1 border border-zinc-200 rounded-md px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-black bg-white">
                    <option value="">Item…</option>
                    {items.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
                  </select>
                  <input type="text" placeholder="Qty" value={inp.quantity} onChange={(e) => setInput(i, "quantity", e.target.value)}
                    className="w-20 border border-zinc-200 rounded-md px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-black" />
                  <input type="text" placeholder="Unit" value={inp.unit} onChange={(e) => setInput(i, "unit", e.target.value)}
                    className="w-16 border border-zinc-200 rounded-md px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-black" />
                  <button type="button" onClick={() => removeInput(i)} className="text-zinc-400 hover:text-black text-xs py-1.5">×</button>
                </div>
              ))}
            </div>
          </div>

          {error && <p className="text-xs text-red-600">{error}</p>}

          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm text-zinc-600 hover:text-black">Cancel</button>
            <button type="submit" disabled={createRecipe.isPending}
              className="bg-black text-white text-sm font-medium px-4 py-2 rounded-md hover:bg-zinc-800 disabled:opacity-50">
              {createRecipe.isPending ? "Creating…" : "Create"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
