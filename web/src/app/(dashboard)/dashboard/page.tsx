const kpis = [
  { label: "Inventory Value", value: "—", sub: "across all locations" },
  { label: "Active Jobs", value: "—", sub: "in production" },
  { label: "Low Stock Alerts", value: "—", sub: "below reorder point" },
  { label: "Recipes Defined", value: "—", sub: "active versions" },
];

export default function DashboardPage() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-black">Overview</h2>
        <p className="text-sm text-zinc-500">Operations summary</p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
        {kpis.map((kpi) => (
          <div key={kpi.label} className="bg-white border border-zinc-200 rounded-lg p-5">
            <p className="text-xs font-medium text-zinc-500 uppercase tracking-wider">{kpi.label}</p>
            <p className="text-3xl font-bold text-black mt-2">{kpi.value}</p>
            <p className="text-xs text-zinc-400 mt-1">{kpi.sub}</p>
          </div>
        ))}
      </div>

      <div className="bg-white border border-zinc-200 rounded-lg p-5">
        <p className="text-sm font-medium text-black mb-4">Recent Activity</p>
        <p className="text-sm text-zinc-400">No activity yet.</p>
      </div>
    </div>
  );
}
