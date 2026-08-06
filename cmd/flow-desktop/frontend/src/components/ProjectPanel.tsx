import type { ProjectDetail } from "../api";
import { useUI } from "../ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { HillTrack } from "./HillTrack";

export function ProjectPanel({ detail, onClose }: { detail: ProjectDetail; onClose: () => void }) {
  const { openDoc } = useUI();
  const cycles = detail.cycles ?? [];
  const specs = detail.specs ?? [];
  const decisions = detail.decisions ?? [];
  const pitches = detail.pitches ?? [];

  const view = (kind: string, id: string, label: string) => () =>
    openDoc({ project: detail.projectId, kind, id, label });

  return (
    <section className="mb-8 rounded-xl border bg-card/40 p-5">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-primary">{detail.name}</h2>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span>{detail.projectId}</span>
            <Badge>baseline: {detail.baseline || "?"}</Badge>
            <Badge>{(detail.errors ?? []).length} index errors</Badge>
            <span className="font-mono">{detail.root}</span>
          </div>
        </div>
        <Button variant="outline" size="sm" onClick={onClose}>
          close
        </Button>
      </div>

      <Tabs defaultValue="cycles">
        <TabsList>
          <TabsTrigger value="cycles">Cycles ({cycles.length})</TabsTrigger>
          <TabsTrigger value="specs">Specs ({specs.length})</TabsTrigger>
          <TabsTrigger value="decisions">ADRs ({decisions.length})</TabsTrigger>
          <TabsTrigger value="pitches">Pitches ({pitches.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="cycles" className="flex flex-col gap-2">
          {cycles.length === 0 && <Empty />}
          {cycles.map((c) => (
            <Card key={c.id} className="cursor-pointer hover:border-primary" onClick={view("cycle", c.id, c.id)}>
              <CardHeader className="p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <strong>{c.id}</strong>
                  <Badge tone={c.status}>{c.status}</Badge>
                  <Badge>bets {(c.bets ?? []).length}/{c.capacity}</Badge>
                  <span className="text-xs text-muted-foreground">
                    {c.starts} → {c.ends} ({c.appetiteWeeks}w)
                  </span>
                </div>
              </CardHeader>
              <CardContent className="p-3 pt-0 text-sm">
                {(c.bets ?? []).map((b) => (
                  <div key={b.lineage} className="flex items-center gap-2 py-0.5">
                    <span className="font-mono text-xs text-muted-foreground">{b.lineage}</span>
                    <Badge tone={b.appetite}>{b.appetite}</Badge>
                    <span className="text-xs text-muted-foreground">placed {b.placed}</span>
                  </div>
                ))}
                {(c.shelved ?? []).map((s) => (
                  <div key={s} className="flex items-center gap-2 py-0.5 opacity-60">
                    <span className="font-mono text-xs text-muted-foreground">{s}</span>
                    <Badge tone="shelved">shelved</Badge>
                  </div>
                ))}
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        <TabsContent value="specs" className="flex flex-col gap-2">
          {specs.length === 0 && <Empty />}
          {specs.map((s) => (
            <Card key={s.lineage} className="cursor-pointer hover:border-primary" onClick={view("spec", s.lineage, s.lineage)}>
              <CardHeader className="p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <strong>{s.lineage}</strong>
                  <Badge tone={s.status}>{s.status}</Badge>
                  <span className="text-xs text-muted-foreground">cycle {s.cycle}</span>
                </div>
              </CardHeader>
              <CardContent className="p-3 pt-0 text-sm">
                {(s.scopes ?? []).map((sc) => (
                  <div key={sc.id} className="flex items-center gap-3 py-1">
                    <span className="min-w-40 font-medium">{sc.title}</span>
                    <Badge tone={sc.status}>{sc.status}</Badge>
                    <HillTrack hill={sc.hill} status={sc.status} size="sm" />
                  </div>
                ))}
                {(s.interfaces ?? []).length > 0 && (
                  <div className="mt-2 text-[11px] uppercase tracking-wide text-muted-foreground">interfaces (locked)</div>
                )}
                {(s.interfaces ?? []).map((i) => (
                  <div key={i.name} className="flex items-center gap-2 py-0.5">
                    <code className="rounded bg-background px-1.5 py-0.5 text-xs text-primary">{i.name}</code>
                    <span className="text-xs text-muted-foreground">{i.contract}</span>
                  </div>
                ))}
                {(s.noGos ?? []).length > 0 && (
                  <>
                    <div className="mt-2 text-[11px] uppercase tracking-wide text-muted-foreground">no-gos</div>
                    <ul className="ml-4 list-disc text-sm">
                      {(s.noGos ?? []).map((n, i) => <li key={i}>{n}</li>)}
                    </ul>
                  </>
                )}
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        <TabsContent value="decisions" className="flex flex-col gap-2">
          {decisions.length === 0 && <Empty />}
          {decisions.map((d) => (
            <Card key={d.id} className="cursor-pointer hover:border-primary" onClick={view("decision", d.id, d.id)}>
              <CardHeader className="p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <strong>{d.id}</strong>
                  <Badge tone={d.layer}>{d.layer}</Badge>
                  <Badge tone={d.status}>{d.status}</Badge>
                  <Badge>{d.reversibility}</Badge>
                </div>
              </CardHeader>
              <CardContent className="p-3 pt-0 text-sm">
                <div>{d.title}</div>
                {d.supersedes && <div className="text-xs text-muted-foreground">supersedes {d.supersedes}</div>}
                {d.supersededBy && <div className="text-xs text-muted-foreground">superseded by {d.supersededBy}</div>}
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        <TabsContent value="pitches" className="flex flex-col gap-2">
          {pitches.length === 0 && <Empty />}
          {pitches.map((p) => (
            <Card key={p.lineage} className="cursor-pointer hover:border-primary" onClick={view("pitch", p.lineage, p.lineage)}>
              <CardHeader className="p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <strong>{p.title}</strong>
                  <Badge tone={p.status}>{p.status}</Badge>
                  <Badge tone={p.appetite}>{p.appetite}</Badge>
                  <span className="font-mono text-xs text-muted-foreground">{p.lineage}</span>
                </div>
              </CardHeader>
              <CardContent className="p-3 pt-0 text-sm">
                {p.problem && <div className="text-muted-foreground">{p.problem}</div>}
                {(p.noGos ?? []).length > 0 && (
                  <>
                    <div className="mt-2 text-[11px] uppercase tracking-wide text-muted-foreground">no-gos</div>
                    <ul className="ml-4 list-disc">{(p.noGos ?? []).map((n, i) => <li key={i}>{n}</li>)}</ul>
                  </>
                )}
                {(p.rabbitHoles ?? []).length > 0 && (
                  <>
                    <div className="mt-2 text-[11px] uppercase tracking-wide text-muted-foreground">rabbit holes</div>
                    <ul className="ml-4 list-disc">{(p.rabbitHoles ?? []).map((n, i) => <li key={i}>{n}</li>)}</ul>
                  </>
                )}
              </CardContent>
            </Card>
          ))}
        </TabsContent>
      </Tabs>
    </section>
  );
}

function Empty() {
  return <div className="text-sm italic text-muted-foreground">none</div>;
}
