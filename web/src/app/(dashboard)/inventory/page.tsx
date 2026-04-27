"use client";

import { useState } from "react";
import { useItems, useCreateItem, type Item } from "@/lib/api/inventory";

export default function InventoryPage() {
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const { data: items = [], isLoading } = useItems({ q: search || undefined });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-black">Inventory</h2>
          <p className="text-sm text-zinc-500">Items and ingredients</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="bg-black text-white text-sm font-medium px-4 py-2 rounded-md hover:bg-zinc-800 transition-colors"
        >
          Add Item
        </button>
      </div>

      <div className="flex gap-3">
        <input
          type="search"
          placeholder="Search by name or SKU…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="border border-zinc-200 rounded-md px-3 py-2 text-sm w-72 focus:outline-none focus:ring-1 focus:ring-black"
        />
      </div>

      <div className="bg-white border border-zinc-200 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-200 bg-zinc-50">
              {["SKU", "Name", "Unit", "Category", "Min Stock", "Status"].map((col) => (
                <th key={col} className="text-left px-4 py-3 text-xs font-medium text-zinc-500 uppercase tracking-wider">
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-sm text-zinc-400">Loading…</td></tr>
            ) : items.length === 0 ? (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-sm text-zinc-400">No items yet.</td></tr>
            ) : (
              items.map((item) => <ItemRow key={item.id} item={item} />)
            )}
          </tbody>
        </table>
      </div>

      {showCreate && <CreateItemDialog onClose={() => setShowCreate(false)} />}
    </div>
  );
}

function ItemRow({ item }: { item: Item }) {
  return (
    <tr className="border-b border-zinc-100 last:border-0 hover:bg-zinc-50">
      <td className="px-4 py-3 font-mono text-xs text-zinc-600">{item.sku}</td>
      <td className="px-4 py-3 font-medium text-black">{item.name}</td>
      <td className="px-4 py-3 text-zinc-600">{item.unit}</td>
      <td className="px-4 py-3 text-zinc-500">{item.category ?? "—"}</td>
      <td className="px-4 py-3 text-zinc-500">{item.min_stock_qty ?? "—"}</td>
      <td className="px-4 py-3">
        <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
          item.is_active ? "bg-black text-white" : "bg-zinc-100 text-zinc-500"
        }`}>
          {item.is_active ? "Active" : "Inactive"}
        </span>
      </td>
    </tr>
  );
}

function CreateItemDialog({ onClose }: { onClose: () => void }) {
  const createItem = useCreateItem();
  const [form, setForm] = useState({ sku: "", name: "", unit: "", category: "", description: "", min_stock_qty: "" });
  const [error, setError] = useState("");

  function set(field: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement>) => setForm((f) => ({ ...f, [field]: e.target.value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await createItem.mutateAsync({
        sku: form.sku,
        name: form.name,
        unit: form.unit,
        category: form.category || undefined,
        description: form.description || undefined,
        min_stock_qty: form.min_stock_qty || undefined,
      });
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to create item");
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-white rounded-lg w-full max-w-md p-6 shadow-xl" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-base font-semibold text-black mb-5">New Item</h3>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <Field label="SKU *" value={form.sku} onChange={set("sku")} required />
            <Field label="Unit *" value={form.unit} onChange={set("unit")} placeholder="kg / L / pcs" required />
          </div>
          <Field label="Name *" value={form.name} onChange={set("name")} required />
          <div className="grid grid-cols-2 gap-3">
            <Field label="Category" value={form.category} onChange={set("category")} />
            <Field label="Min Stock Qty" value={form.min_stock_qty} onChange={set("min_stock_qty")} placeholder="0.00" />
          </div>
          <Field label="Description" value={form.description} onChange={set("description")} />
          {error && <p className="text-xs text-red-600">{error}</p>}
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm text-zinc-600 hover:text-black">Cancel</button>
            <button
              type="submit"
              disabled={createItem.isPending}
              className="bg-black text-white text-sm font-medium px-4 py-2 rounded-md hover:bg-zinc-800 disabled:opacity-50"
            >
              {createItem.isPending ? "Creating…" : "Create"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function Field({ label, value, onChange, placeholder, required }: {
  label: string; value: string;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  placeholder?: string; required?: boolean;
}) {
  return (
    <div>
      <label className="block text-xs font-medium text-zinc-700 mb-1">{label}</label>
      <input
        type="text"
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        required={required}
        className="w-full border border-zinc-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-black"
      />
    </div>
  );
}
