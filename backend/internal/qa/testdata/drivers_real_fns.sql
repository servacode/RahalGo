-- Test oracle ONLY (TestRealDrive_EqualsValidatedSQL) — never a second production implementation.
-- Copied verbatim from the validated report D:/tmp/drivers_real.sql (owner decision 2026-11-05):
-- only its two CREATE FUNCTION blocks; the \set line and the report queries are dropped.
-- driving = segment speed >= 3 m/s (10.8 km/h), phone speed not below 1.5 m/s when the phone reports it,
--           and both points with accuracy <= 30 m. Everything else that the page counts = slow or drift.
-- Temporary functions (pg_temp) that vanish when the session ends.

CREATE FUNCTION pg_temp.rg_hav(a1 float8, o1 float8, a2 float8, o2 float8) RETURNS float8
LANGUAGE sql IMMUTABLE AS $f$
  SELECT 2 * 6371000.0 * atan2(sqrt(h), sqrt(1 - h))
  FROM (SELECT sin(radians(a2 - a1) / 2) ^ 2
             + cos(radians(a1)) * cos(radians(a2)) * sin(radians(o2 - o1) / 2) ^ 2 AS h) s
$f$;

CREATE FUNCTION pg_temp.rg_real(did uuid, t0 timestamptz, t1 timestamptz,
                              OUT km_total float8, OUT km_with float8, OUT km_to float8, OUT km_no float8,
                              OUT drive_orders float8, OUT drive_no float8, OUT drive_no_min float8, OUT km_slow float8)
LANGUAGE plpgsql AS $f$
DECLARE
  r record;
  n int := 0;
  ts timestamptz[] := '{}';
  la float8[] := '{}';
  lo float8[] := '{}';
  ac float8[] := '{}';
  sp float8[] := '{}';
  st int[] := '{}';
  wa timestamptz[] := '{}';
  wp timestamptz[] := '{}';
  we timestamptz[] := '{}';
  nw int := 0;
  i int; j int; k int; q int; sid int := 0;
  dt float8; d float8; mid timestamptz; e timestamptz;
  w_with boolean; w_to boolean; drive boolean;
BEGIN
  km_with := 0; km_to := 0; km_no := 0;
  drive_orders := 0; drive_no := 0; drive_no_min := 0; km_slow := 0;

  FOR r IN
    SELECT COALESCE((SELECT min(ev.created_at) FROM order_events ev
                      WHERE ev.order_id = o.id AND ev.driver_id = did AND ev.to_status = 'assigned'),
                    o.accepted_at, o.dispatched_at, o.picked_up_at) AS a,
           o.picked_up_at AS p,
           COALESCE(o.delivered_at, o.returned_at, o.closed_at) AS e
      FROM orders o
     WHERE o.driver_id = did
       AND COALESCE(o.picked_up_at, o.accepted_at, o.dispatched_at, o.created_at) < t1
       AND COALESCE(o.delivered_at, o.returned_at, o.closed_at, t1) > t0
     ORDER BY o.created_at LOOP
    nw := nw + 1;
    wa[nw] := r.a; wp[nw] := r.p; we[nw] := COALESCE(r.e, t1);
  END LOOP;

  FOR r IN SELECT recorded_at, lat, lng, accuracy_m, speed_mps FROM driver_trail
            WHERE driver_id = did AND recorded_at >= t0 AND recorded_at < t1
            ORDER BY recorded_at LOOP
    IF r.accuracy_m IS NOT NULL AND r.accuracy_m > 50 THEN CONTINUE; END IF;
    IF n > 0 THEN
      dt := extract(epoch FROM r.recorded_at - ts[n]);
      d := pg_temp.rg_hav(la[n], lo[n], r.lat, r.lng);
      IF dt <= 0 AND d > 0 THEN CONTINUE; END IF;
      IF dt > 0 AND d / dt > 120 / 3.6 AND d > 100 THEN CONTINUE; END IF;
    END IF;
    n := n + 1;
    ts[n] := r.recorded_at; la[n] := r.lat; lo[n] := r.lng; st[n] := 0;
    ac[n] := r.accuracy_m; sp[n] := r.speed_mps;
  END LOOP;

  i := 1;
  WHILE i <= n LOOP
    j := i;
    WHILE j + 1 <= n
      AND pg_temp.rg_hav(la[i], lo[i], la[j + 1], lo[j + 1]) <= 40
      AND ts[j + 1] - ts[j] <= interval '6 minutes' LOOP
      j := j + 1;
    END LOOP;
    IF j > i AND ts[j] - ts[i] >= interval '3 minutes' THEN
      sid := sid + 1;
      FOR k IN i..j LOOP st[k] := sid; END LOOP;
      i := j + 1;
    ELSE
      i := i + 1;
    END IF;
  END LOOP;

  FOR k IN 2..n LOOP
    IF ts[k] - ts[k - 1] > interval '6 minutes' THEN CONTINUE; END IF;
    IF st[k] > 0 AND st[k] = st[k - 1] THEN CONTINUE; END IF;
    d := pg_temp.rg_hav(la[k - 1], lo[k - 1], la[k], lo[k]);
    dt := extract(epoch FROM ts[k] - ts[k - 1]);
    mid := ts[k - 1] + (ts[k] - ts[k - 1]) / 2;
    w_with := false; w_to := false;
    FOR q IN 1..nw LOOP
      e := we[q];
      IF mid > e THEN CONTINUE; END IF;
      IF wp[q] IS NOT NULL AND mid >= wp[q] THEN
        w_with := true;
        CONTINUE;
      END IF;
      IF wa[q] IS NOT NULL AND mid >= wa[q] THEN
        w_to := true;
      END IF;
    END LOOP;
    IF w_with THEN km_with := km_with + d / 1000;
    ELSIF w_to THEN km_to := km_to + d / 1000;
    ELSE km_no := km_no + d / 1000;
    END IF;
    drive := dt > 0 AND d / dt >= 3
             AND (sp[k] IS NULL OR sp[k] >= 1.5)
             AND COALESCE(ac[k - 1], 0) <= 30 AND COALESCE(ac[k], 0) <= 30;
    IF drive THEN
      IF w_with OR w_to THEN drive_orders := drive_orders + d / 1000;
      ELSE
        drive_no := drive_no + d / 1000;
        drive_no_min := drive_no_min + dt / 60;
      END IF;
    ELSE
      km_slow := km_slow + d / 1000;
    END IF;
  END LOOP;
  km_total := km_with + km_to + km_no;
END
$f$;
