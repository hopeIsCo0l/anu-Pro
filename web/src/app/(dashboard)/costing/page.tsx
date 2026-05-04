const columns = ["Item", "Cost Method", "Unit Cost", "Currency", "Last Updated"];

export default function CostingPage() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-black">Costing</h2>
        <p className="text-sm text-zinc-500">Weighted average unit costs and history</p>
      </div>

      <div className="bg-white border border-zinc-200 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-200 bg-zinc-50">
              {columns.map((col) => (
                <th key={col} className="text-left px-4 py-3 text-xs font-medium text-zinc-500 uppercase tracking-wider">
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            <tr>
              <td colSpan={columns.length} className="px-4 py-8 text-center text-sm text-zinc-400">
                No cost data yet.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  );
}
