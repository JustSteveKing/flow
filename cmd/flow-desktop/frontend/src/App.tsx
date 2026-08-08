import { useCallback, useEffect, useRef, useState } from "react";
import { api, type DocRef, type DocView, type ProjectDetail, type Snapshot } from "./api";
import { UIContext } from "./ui";
import { Sidebar } from "./components/Sidebar";
import { BettingTable } from "./components/BettingTable";
import { PortfolioHill } from "./components/PortfolioHill";
import { Supersession } from "./components/Supersession";
import { TaskBoard } from "./components/TaskBoard";
import { ProjectPanel } from "./components/ProjectPanel";
import { DocModal } from "./components/DocModal";
import { Toast } from "./components/Toast";

const POLL_MS = 1500;

export default function App() {
  const [snap, setSnap] = useState<Snapshot | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [detail, setDetail] = useState<ProjectDetail | null>(null);
  const [doc, setDoc] = useState<{ ref: DocRef; view: DocView | null } | null>(null);
  const [toastMsg, setToastMsg] = useState<string | null>(null);

  const selectedRef = useRef<string | null>(null);
  selectedRef.current = selected;

  const loadSnapshot = useCallback(async () => {
    try {
      setSnap(await api.snapshot());
    } catch {
      /* transient during reindex */
    }
  }, []);

  const loadDetail = useCallback(async () => {
    const id = selectedRef.current;
    if (!id) return;
    try {
      setDetail(await api.project(id));
    } catch {
      /* project may have gone away */
    }
  }, []);

  const refresh = useCallback(() => {
    loadSnapshot();
    loadDetail();
  }, [loadSnapshot, loadDetail]);

  useEffect(() => {
    loadSnapshot();
    const t = setInterval(() => {
      loadSnapshot();
      loadDetail();
    }, POLL_MS);
    return () => clearInterval(t);
  }, [loadSnapshot, loadDetail]);

  useEffect(() => {
    setDetail(null);
    loadDetail();
  }, [selected, loadDetail]);

  const toast = useCallback((message: string) => {
    setToastMsg(message);
    window.setTimeout(() => setToastMsg(null), 3500);
  }, []);

  const openDoc = useCallback(async (ref: DocRef) => {
    setDoc({ ref, view: null });
    try {
      const view = await api.doc(ref.project, ref.kind, ref.id);
      setDoc({ ref, view });
    } catch {
      toast(`could not load ${ref.kind} ${ref.id}`);
      setDoc(null);
    }
  }, [toast]);

  return (
    <UIContext.Provider value={{ openDoc, toast, refresh }}>
      <div className="grid h-screen grid-cols-[260px_1fr]">
        <Sidebar
          projects={snap?.projects ?? []}
          selected={selected}
          onSelect={(id) => setSelected((cur) => (cur === id ? null : id))}
        />

        <main className="overflow-y-auto p-7">
          {selected && detail && (
            <ProjectPanel detail={detail} onClose={() => setSelected(null)} />
          )}

          <Section title="Task board">
            <TaskBoard tasks={snap?.tasks ?? []} />
          </Section>

          <Section title="Unified betting table">
            <BettingTable rows={snap?.bettingTable ?? []} />
          </Section>

          <Section title="Portfolio hill chart">
            <PortfolioHill points={snap?.portfolioHill ?? []} />
          </Section>

          <Section title="ADR supersession graph">
            <Supersession edges={snap?.supersession ?? []} />
          </Section>
        </main>
      </div>

      <DocModal
        open={doc !== null}
        docRef={doc?.ref ?? null}
        view={doc?.view ?? null}
        onClose={() => setDoc(null)}
      />
      <Toast message={toastMsg} />
    </UIContext.Provider>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="mb-7">
      <h2 className="mb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground">
        {title}
      </h2>
      {children}
    </section>
  );
}
