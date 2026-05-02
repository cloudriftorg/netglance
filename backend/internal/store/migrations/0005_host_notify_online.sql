-- Per-host back-online notification flag. Default off so notifications
-- are opt-in per device (settings toggles still gate the global on/off).
-- Also reset notify_offline on every existing row so users get a clean
-- slate aligned with the new defaults — any host that should notify
-- needs the flag re-enabled from the host detail page.
ALTER TABLE hosts ADD COLUMN notify_online INTEGER NOT NULL DEFAULT 0;
UPDATE hosts SET notify_offline = 0;
