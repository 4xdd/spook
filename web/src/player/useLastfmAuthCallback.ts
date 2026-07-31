import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { completeLastfmAuth, consumeLastfmAuthPending } from "@/lib/lastfm";

/**
 * Completes the Last.fm web auth redirect (?token=…) and opens Settings so the
 * user sees the connected account.
 */
export function useLastfmAuthCallback(onConnected: () => void): string | null {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const token = params.get("token");
    if (!token) return;
    if (!consumeLastfmAuthPending()) {
      // Stray token in the URL — strip it without treating it as auth.
      navigate({ pathname: "/", search: "" }, { replace: true });
      return;
    }

    let cancelled = false;
    void completeLastfmAuth(token)
      .then(() => {
        if (cancelled) return;
        navigate({ pathname: "/", search: "" }, { replace: true });
        onConnected();
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        navigate({ pathname: "/", search: "" }, { replace: true });
        setError(err instanceof Error ? err.message : "Last.fm login failed");
        onConnected();
      });

    return () => {
      cancelled = true;
    };
  }, [navigate, onConnected, params]);

  return error;
}
