-- The scans table only ever served the "Last scan" badge on the Hosts page,
-- which needs a single timestamp + hostsFound. A full append-only table
-- (with zombie rows on crash and unbounded growth) was overkill for that —
-- the data now lives as a single JSON value under the `lastScan` settings
-- key.
DROP TABLE IF EXISTS scans;
