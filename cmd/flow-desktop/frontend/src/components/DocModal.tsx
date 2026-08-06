import type { DocRef, DocView } from "../api";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Markdown } from "./Markdown";

export function DocModal({
  open,
  docRef,
  view,
  onClose,
}: {
  open: boolean;
  docRef: DocRef | null;
  view: DocView | null;
  onClose: () => void;
}) {
  const fields = view?.frontmatter ?? [];

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader className="shrink-0">
          <DialogTitle className="truncate pr-8 font-mono text-xs text-muted-foreground">
            {view?.path ?? docRef?.label ?? "source"}
          </DialogTitle>
        </DialogHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="p-4">
            {!view && <div className="text-sm text-muted-foreground">loading…</div>}

            {view && fields.length > 0 && (
              <table className="mb-4 w-full border-collapse overflow-hidden rounded-md border text-sm">
                <tbody>
                  {fields.map((f) => (
                    <tr key={f.key} className="border-b last:border-0">
                      <th className="w-44 bg-secondary/50 px-3 py-1.5 text-left align-top font-medium text-muted-foreground">
                        {f.key}
                      </th>
                      <td className="px-3 py-1.5 align-top">
                        {f.value.includes("\n") ? (
                          <pre className="whitespace-pre-wrap font-mono text-[12px] leading-relaxed">{f.value}</pre>
                        ) : (
                          <span className="break-words">{f.value}</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}

            {view && view.body.trim() && (
              <div className="text-sm">
                <Markdown>{view.body}</Markdown>
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
