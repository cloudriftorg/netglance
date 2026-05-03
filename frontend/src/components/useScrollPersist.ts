import { useCallback, useEffect, useRef } from 'react';

// useScrollPersist returns a callback ref to attach to a scrollable
// element. On mount it restores the previously saved scrollTop for
// the given key; on every scroll it stores the new position back.
//
// Why a callback ref and not a useEffect+useRef pair: the scrollable
// element is often conditionally rendered (e.g. Hosts shows a
// placeholder while the host list is empty, then swaps it for the
// real list once data arrives). With a useEffect the setup only runs
// at component mount — by that time the placeholder is on screen and
// `ref.current` is null, so the effect bails. The callback ref
// instead fires *whenever the element appears or disappears*, so the
// observers attach exactly when the real scrollable container shows
// up.
//
// Storage is sessionStorage so the position survives in-app
// navigations (the page unmounts when you visit /settings and
// remounts on /hosts) but resets when the browser tab closes — the
// right granularity for "remember where I was while I'm using the
// app".
//
// Restore is retried via ResizeObserver + MutationObserver until the
// container is tall enough to honour the saved value (the host list
// fetch is async, so the initial render has scrollHeight=0). Once
// restored, or once the user scrolls themselves, we stop retrying.
export function useScrollPersist<T extends HTMLElement>(key: string) {
  // Bag of mutable state that survives across the callback-ref's
  // attach / detach cycles for this key. Each new ref invocation
  // creates fresh observers + restore state.
  const teardownRef = useRef<(() => void) | null>(null);

  const setup = useCallback(
    (el: T | null) => {
      // Element detached (or replaced): tear down previous observers.
      if (teardownRef.current) {
        teardownRef.current();
        teardownRef.current = null;
      }
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
        const max = el.scrollHeight - el.clientHeight;
        if (max >= saved) {
          suppressNextScroll = true;
          el.scrollTop = saved;
          restored = true;
        }
      };

      tryRestore();

      const ro = new ResizeObserver(tryRestore);
      ro.observe(el);
      const mo = new MutationObserver(tryRestore);
      mo.observe(el, { childList: true, subtree: true });

      let frame: number | null = null;
      const onScroll = () => {
        if (suppressNextScroll) {
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

      teardownRef.current = () => {
        el.removeEventListener('scroll', onScroll);
        ro.disconnect();
        mo.disconnect();
        if (frame != null) cancelAnimationFrame(frame);
      };
    },
    [key],
  );

  // Cleanup on unmount as a backstop in case React doesn't call the
  // callback ref with null (it should, but being explicit is cheap).
  useEffect(() => {
    return () => {
      if (teardownRef.current) {
        teardownRef.current();
        teardownRef.current = null;
      }
    };
  }, []);

  return setup;
}
