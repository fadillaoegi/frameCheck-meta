import { createContext, useContext } from "react";
import type { UseCases } from "../../application/usecases";

export const UseCasesContext = createContext<UseCases | null>(null);

export function useUseCases(): UseCases {
  const uc = useContext(UseCasesContext);
  if (!uc) throw new Error("UseCasesContext is not provided");
  return uc;
}
