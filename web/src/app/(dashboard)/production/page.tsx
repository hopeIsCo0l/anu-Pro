const columns = ["Job #", "Recipe", "Status", "Started", "Expected Output", "Yield"];

export default function ProductionPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-black">Production</h2>
          <p className="text-sm text-zinc-500">Jobs, lot consumption, and outputs</p>
        </div>
        <button className="bg-black text-white text-sm font-medium px-4 py-2 rounded-md hover:bg-zinc-800 transition-colors">
          New Job
        </button>
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
                No production jobs yet.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  );
}
