import { useEffect, useRef } from 'react';

// useScrollPersist returns a ref to attach to a scrollable element.
// On mount it restores the previously saved scrollTop for the given
// key; on every scroll it stores the new position back. Storage is
// sessionStorage so the position survives in-app navigations (the
// page unmounts when you visit /settings and remounts on /hosts) but
// resets when the browser tab closes — which is the right granularity
// for "remember where I was while I'm using the app".
//
// The tricky part is *when* to restore. The host list arrives async
// (initial fetch hasn't completed when the component mounts), so a
// naive `scrollTop = saved` runs against an empty container and
// silently clamps to 0. We instead watch the container with a
// ResizeObserver + MutationObserver and re-attempt restore on every
// content change until either:
//   • the scrollable area is tall enough to honour the saved value, or
//   • the user starts scrolling themselves (don't fight them).
export function useScrollPersist<T extends HTMLElement>(key: string) {
  const ref = useRef<T | null>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const storageKey = `netglance.scroll.${key}`;

    let saved: number | null = null;
    try {
      const raw = sessionStorage.getItem(storageKey);
      if (raw != null) {
        const n = Number(raw);
        if (Number.isFinite(n)) saved = n;
      }
    } catch {
      /* storage disabled — silent */
    }

    let restored = saved == null || saved === 0;
    let userScrolled = false;
    let suppressNextScroll = false;

    const tryRestore = () => {
      if (restored || userScrolled || saved == null) return;
      // Only restore once the container can actually hold the saved
      // scrollTop — otherwise the assignment clamps to whatever the
      // current max is, and we lose the saved value forever the
      // moment the user scrolls (which writes the clamped value
      // back).
      const max = el.scrollHeight - el.clientHeight;
      if (max >= saved) {
        suppressNextScroll = true;
        el.scrollTop = saved;
        restored = true;
      }
    };

    // First synchronous attempt — handles the case where the page
    // didn't have to load anything (cached / instant render).
    tryRestore();

    const ro = new ResizeObserver(tryRestore);
    ro.observe(el);
    const mo = new MutationObserver(tryRestore);
    mo.observe(el, { childList: true, subtree: true });

    let frame: number | null = null;
    const onScroll = () => {
      if (suppressNextScroll) {
        // Programmatic restore fires a scroll event; ignore it so we
        // don't immediately flip into "user is scrolling" mode and
        // disable future re-attempts.
        suppressNextScroll = false;
        return;
      }
      userScrolled = true;
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
      ro.disconnect();
      mo.disconnect();
      if (frame != null) cancelAnimationFrame(frame);
    };
  }, [key]);

  return ref;
}
