import type { ProjectSummary } from "../api";
import { cn } from "@/lib/utils";

export function Sidebar({
  projects,
  selected,
  onSelect,
}: {
  projects: ProjectSummary[];
  selected: string | null;
  onSelect: (id: string) => void;
}) {
  return (
    <aside className="flex flex-col overflow-y-auto border-r bg-card p-5">
      <h1 className="text-2xl font-semibold tracking-tight">flow</h1>
      <p className="mb-6 text-xs text-muted-foreground">workspace overlay</p>

      <h2 className="mb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground">
        projects
      </h2>
      <ul className="flex flex-col gap-1">
        {projects.length === 0 && (
          <li className="text-sm italic text-muted-foreground">no projects registered</li>
        )}
        {projects.map((p) => (
          <li
            key={p.projectId}
            onClick={() => onSelect(p.projectId)}
            className={cn(
              "cursor-pointer rounded-md bg-secondary/50 p-2.5 transition-colors hover:bg-secondary",
              selected === p.projectId && "bg-accent/60 outline outline-1 outline-primary"
            )}
          >
            <div className="font-medium">{p.projectId}</div>
            <div className="text-xs text-muted-foreground">
              {p.name}
              {p.errors > 0 && <span className="ml-2 text-destructive">{p.errors} err</span>}
              {p.warnings > 0 && <span className="ml-2 text-uphill">{p.warnings} warn</span>}
              {p.errors === 0 && p.warnings === 0 && <span className="ml-2">clean</span>}
            </div>
          </li>
        ))}
      </ul>

      <p className="mt-auto pt-6 text-[11px] leading-relaxed text-muted-foreground">
        A lens over the files. Truth lives in each project's git.
      </p>
    </aside>
  );
}
