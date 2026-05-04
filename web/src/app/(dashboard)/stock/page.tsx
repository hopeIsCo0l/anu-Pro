"use client";

import { useCurrentStock } from "@/lib/api/inventory";

export default function StockPage() {
  const { data: stock = [], isLoading } = useCurrentStock();

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-black">Stock</h2>
        <p className="text-sm text-zinc-500">Current on-hand quantities</p>
      </div>

      <div className="bg-white border border-zinc-200 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-200 bg-zinc-50">
              {["Item", "SKU", "Lot", "Location", "Qty", "Unit"].map((col) => (
                <th key={col} className="text-left px-4 py-3 text-xs font-medium text-zinc-500 uppercase tracking-wider">
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-sm text-zinc-400">Loading…</td></tr>
            ) : stock.length === 0 ? (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-sm text-zinc-400">No stock movements recorded yet.</td></tr>
            ) : (
              stock.map((row, i) => (
                <tr key={i} className="border-b border-zinc-100 last:border-0 hover:bg-zinc-50">
                  <td className="px-4 py-3 font-medium text-black">{row.item_name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-zinc-500">{row.item_sku}</td>
                  <td className="px-4 py-3 text-zinc-600">{row.lot_number}</td>
                  <td className="px-4 py-3 text-zinc-600">{row.location_name} <span className="text-zinc-400">({row.location_code})</span></td>
                  <td className="px-4 py-3 font-medium tabular-nums">{row.quantity}</td>
                  <td className="px-4 py-3 text-zinc-500">{row.unit}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
