import { createContext, useContext } from "react";
import type { DocRef } from "./api";

// Cross-cutting UI actions shared by the views: open a document's source, show a
// transient error, and force a data refresh after a write.
export interface UIActions {
  openDoc: (ref: DocRef) => void;
  toast: (message: string) => void;
  refresh: () => void;
}

export const UIContext = createContext<UIActions>({
  openDoc: () => {},
  toast: () => {},
  refresh: () => {},
});

export const useUI = () => useContext(UIContext);
