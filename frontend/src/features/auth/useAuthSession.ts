import { useCallback, useEffect, useState } from "preact/hooks";
import { getSession } from "../../lib/auth";
import type { AuthSessionState } from "./types";

export function useAuthSession() {
  const [state, setState] = useState<AuthSessionState>({ status: "loading" });

  const refresh = useCallback(() => {
    let isCurrent = true;

    setState({ status: "loading" });

    getSession().then((result) => {
      if (!isCurrent) {
        return;
      }

      if (result.status === "authenticated") {
        setState({ status: "authenticated" });
        return;
      }

      if (result.status === "unauthenticated") {
        setState({ status: "unauthenticated" });
        return;
      }

      setState({ status: "unavailable" });
    });

    return () => {
      isCurrent = false;
    };
  }, []);

  useEffect(() => refresh(), [refresh]);

  return {
    markAuthenticated: () => setState({ status: "authenticated" }),
    markUnauthenticated: () => setState({ status: "unauthenticated" }),
    refresh,
    state
  };
}
