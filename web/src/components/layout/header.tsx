"use client";

import { usePathname } from "next/navigation";

const titles: Record<string, string> = {
  "/dashboard": "Dashboard",
  "/inventory": "Inventory",
  "/stock": "Stock",
  "/recipes": "Recipes",
  "/production": "Production",
  "/costing": "Costing",
  "/settings": "Settings",
};

export function Header() {
  const pathname = usePathname();
  const title = Object.entries(titles).find(([k]) => pathname.startsWith(k))?.[1] ?? "Anu Pro";

  return (
    <header className="h-14 border-b border-zinc-200 bg-white flex items-center justify-between px-6 flex-shrink-0">
      <span className="text-sm font-medium text-black">{title}</span>
      <div className="w-7 h-7 rounded-full bg-black text-white text-xs flex items-center justify-center font-medium select-none">
        U
      </div>
    </header>
  );
}
