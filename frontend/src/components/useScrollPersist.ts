import { useEffect, useRef } from 'react';

// useScrollPersist returns a ref to attach to a scrollable element.
// On mount it restores the previously saved scrollTop for the given
// key; on every scroll it stores the new position back. Storage is
// sessionStorage so the position survives in-app navigations (the
// page unmounts when you visit /settings and remounts on /hosts) but
// resets when the browser tab closes — which is the right granularity
// for "remember where I was while I'm using the app".
//
// Throttled via requestAnimationFrame so a fast scroll doesn't write
// dozens of times per second.
export function useScrollPersist<T extends HTMLElement>(key: string) {
  const ref = useRef<T | null>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const storageKey = `netglance.scroll.${key}`;

    // Restore saved position. We defer to the next frame so any
    // late-arriving content (e.g. the host list after the first
    // fetch) has had a chance to push the scroll height up before
    // we set scrollTop — otherwise a saved 800px on a still-empty
    // element silently clamps to 0.
    requestAnimationFrame(() => {
      try {
        const raw = sessionStorage.getItem(storageKey);
        if (raw == null) return;
        const top = Number(raw);
        if (Number.isFinite(top)) el.scrollTop = top;
      } catch {
        /* storage disabled — silent */
      }
    });

    let frame: number | null = null;
    const onScroll = () => {
      if (frame != null) return;
      frame = requestAnimationFrame(() => {
        frame = null;
        try {
          sessionStorage.setItem(storageKey, String(el.scrollTop));
        } catch {
          /* storage disabled / quota — silent */
        }
      });
    };
    el.addEventListener('scroll', onScroll, { passive: true });
    return () => {
      el.removeEventListener('scroll', onScroll);
      if (frame != null) cancelAnimationFrame(frame);
    };
  }, [key]);

  return ref;
}
