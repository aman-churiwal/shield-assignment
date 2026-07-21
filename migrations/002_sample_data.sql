-- 2026-07-21: three unique users, one of them logging in twice (duplicate).
INSERT INTO user_logins (user_id, login_time)
VALUES ('11111111-1111-1111-1111-111111111111', '2026-07-21T08:00:00Z'),
       ('22222222-2222-2222-2222-222222222222', '2026-07-21T09:30:00Z'),
       ('33333333-3333-3333-3333-333333333333', '2026-07-21T18:45:00Z'),
       -- exact duplicate of the row above: same user_id AND same login_time.
       -- ON CONFLICT DO NOTHING means this does not inflate the unique count.
       ('33333333-3333-3333-3333-333333333333', '2026-07-21T18:45:00Z') ON CONFLICT (user_id, login_time) DO NOTHING;
-- Expected: GetDailyUniqueUsers("2026-07-21", "UTC") -> 3

-- Day boundary: one login just before midnight UTC on Jul 21, one just after
-- midnight (now Jul 22). These must land in different daily buckets.
INSERT INTO user_logins (user_id, login_time)
VALUES ('44444444-4444-4444-4444-444444444444', '2026-07-21T23:59:59Z'),
       ('55555555-5555-5555-5555-555555555555', '2026-07-22T00:00:01Z') ON CONFLICT (user_id, login_time) DO NOTHING;
-- Expected: GetDailyUniqueUsers("2026-07-21", "UTC") -> 4 (3 above + user 44444444)
-- Expected: GetDailyUniqueUsers("2026-07-22", "UTC") -> 1 (user 55555555)

-- Same month: Jul 21 vs Jul 22.
INSERT INTO user_logins (user_id, login_time)
VALUES ('11111111-1111-1111-1111-111111111111', '2026-07-21T22:00:00Z'),
       ('22222222-2222-2222-2222-222222222222', '2026-07-22T01:00:00Z') ON CONFLICT (user_id, login_time) DO NOTHING;
-- Expected: GetMonthlyUniqueUsers("2026-07", "UTC") includes both users.

-- Day boundary at the start of the period: Jul 20 vs Jul 21.
INSERT INTO user_logins (user_id, login_time)
VALUES ('33333333-3333-3333-3333-333333333333', '2026-07-20T23:30:00Z'),
       ('44444444-4444-4444-4444-444444444444', '2026-07-21T00:30:00Z') ON CONFLICT (user_id, login_time) DO NOTHING;
-- Expected: GetDailyUniqueUsers("2026-07-20", "UTC") -> includes user 33333333
--           GetDailyUniqueUsers("2026-07-21", "UTC") -> includes user 44444444

-- Timezone edge case: 2026-07-21T19:00:00Z is still July 21st in UTC, but
-- it's already July 22nd in India (UTC+5:30, i.e. 2026-07-22T00:30 IST).
INSERT INTO user_logins (user_id, login_time)
VALUES ('55555555-5555-5555-5555-555555555555', '2026-07-21T19:00:00Z') ON CONFLICT (user_id, login_time) DO NOTHING;
-- Expected: GetDailyUniqueUsers("2026-07-21", "UTC")          -> includes user 55555555
-- Expected: GetDailyUniqueUsers("2026-07-21", "Asia/Kolkata") -> does NOT include user 55555555
-- Expected: GetDailyUniqueUsers("2026-07-22", "Asia/Kolkata") -> includes user 55555555