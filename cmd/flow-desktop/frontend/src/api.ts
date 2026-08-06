// Thin client over the in-process /api surface. Types mirror the Go DTOs.

export interface ProjectSummary {
  projectId: string;
  name: string;
  root: string;
  errors: number;
  warnings: number;
}
export interface BetRow {
  projectId: string;
  cycle: string;
  lineage: string;
  title: string;
  appetite: string;
  shelved: boolean;
}
export interface HillPoint {
  projectId: string;
  lineage: string;
  scopeId: string;
  title: string;
  status: string;
  hill: number;
}
export interface Edge {
  projectId: string;
  from: string;
  to: string;
  fromLayer: string;
  toLayer: string;
}
export interface Snapshot {
  projects: ProjectSummary[];
  bettingTable: BetRow[];
  portfolioHill: HillPoint[];
  supersession: Edge[];
}

export interface BetItem { lineage: string; appetite: string; placed: string }
export interface CycleDTO {
  id: string; status: string; appetiteWeeks: number; capacity: number;
  starts: string; ends: string; bets: BetItem[] | null; shelved: string[] | null;
}
export interface InterfaceDTO { name: string; contract: string }
export interface ScopeDTO { id: string; title: string; status: string; hill: number; interfaces: string[] | null }
export interface SpecDTO {
  lineage: string; status: string; cycle: string; noGos: string[] | null;
  assumesDecisions: string[] | null; interfaces: InterfaceDTO[] | null; scopes: ScopeDTO[] | null;
}
export interface DecisionDTO {
  id: string; layer: string; status: string; title: string; reversibility: string;
  lineage: string; supersedes: string; supersededBy: string; decided: string;
}
export interface PitchDTO {
  lineage: string; status: string; title: string; appetite: string; problem: string;
  noGos: string[] | null; assumesDecisions: string[] | null; rabbitHoles: string[] | null;
}
export interface ProjectDetail {
  projectId: string; name: string; root: string; baseline: string;
  cycles: CycleDTO[] | null; specs: SpecDTO[] | null;
  decisions: DecisionDTO[] | null; pitches: PitchDTO[] | null; errors: string[] | null;
}
export interface Field { key: string; value: string }
export interface DocView {
  path: string;
  content: string;
  frontmatter: Field[] | null;
  body: string;
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error ?? res.statusText);
  return res.json();
}

async function postJSON(url: string, body: unknown): Promise<void> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error ?? res.statusText);
}

export const api = {
  snapshot: () => getJSON<Snapshot>("/api/snapshot"),
  project: (id: string) => getJSON<ProjectDetail>(`/api/project?id=${encodeURIComponent(id)}`),
  doc: (project: string, kind: string, id: string) =>
    getJSON<DocView>(`/api/doc?project=${encodeURIComponent(project)}&kind=${kind}&id=${encodeURIComponent(id)}`),
  bet: (projectId: string, lineage: string) => postJSON("/api/bet", { projectId, lineage }),
  shelve: (projectId: string, lineage: string) => postJSON("/api/shelve", { projectId, lineage }),
  hill: (projectId: string, lineage: string, scopeId: string, hill: number) =>
    postJSON("/api/hill", { projectId, lineage, scopeId, hill }),
};

export type DocRef = { project: string; kind: string; id: string; label: string };
