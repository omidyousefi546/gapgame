# GapGame — Full Code Review & Refactor Report

> Reviewed at commit head of `omidyousefi546/gapgame`, Go 1.25, ~11,200 LOC.
> All changes described below are applied in this workspace and the project
> now passes `go build ./...`, `go vet ./...`, and `go test ./...`.

---

## 1. Production readiness verdict: **Not yet — close, but blockers exist(ed)**

### 🔴 Blockers found (some now fixed)

| # | Issue | Status |
|---|-------|--------|
| 1 | **The repo did not compile.** `internal/bot/01_start.go` used `word_guess` without importing it; `10_game.go` imported `zap` unused; `pkg/cleanup/cleanup.go` referenced `rm.Mu / rm.Rooms / rm.PlayerRoom` fields that no longer exist on the Redis-backed `RoomManager`. | ✅ Fixed |
| 2 | **No ban system / no admin tooling.** Anyone reported could keep using the bot. | ✅ Added (`Banned` column + ban guard middleware + admin commands) |
| 3 | **Coin operations are not atomic.** `AwardCoinsByTelegramID` / `DeductCoins` do read-modify-write (`u.Coins += n; Save(u)`). Two concurrent requests can double-spend or lose a refund. Must become `UPDATE users SET coins = coins + ? WHERE coins >= ?` (single SQL) or a transaction. The new admin coin commands already use atomic `gorm.Expr("coins + ?")` updates — migrate the rest the same way. | ⚠️ Documented (behavioral change, left for you) |
| 4 | **Secrets & logging.** `gormLogger.Info` logs every SQL statement (PII: GPS coordinates, ages) — use `Warn`/`Error` in production. `logs/request.log` is committed to git. `.env` handling is fine, but add `.gitignore` for logs. | ⚠️ Documented |
| 5 | **No graceful-shutdown of the poller before workers.** `main.go` cancels ctx and calls `h.Stop()`, but in-flight broadcast/match goroutines aren't awaited (no `sync.WaitGroup`). Low risk, worth fixing. | ⚠️ Documented |

### 🟠 Architecture

* **Good:** clean layering (`bot → service → repository → storage`), Redis for
  ephemeral state, Postgres for durable state, games behind a `GameState`
  interface with factories, middleware chain (recovery → logging → rate-limit).
* **Duplication / dead code:** there were *two* queue systems
  (`internal/matching.QueueManager` — in-memory, **never wired into main.go** —
  and the Redis queue actually used by `match_worker.go`). Same for
  `bot.SearchService` (unused duplicate of the search logic in `05_search.go`),
  `pkg/cleanup/service_cleaner.go` (placeholder no-ops), `internal/storage/sqlite.go`
  and `connection_pool.go` (unused). **Recommendation:** delete the whole
  `internal/matching` package, `search_service.go`, `service_cleaner.go` —
  dead code is a maintenance tax and confuses every reviewer about which queue
  is real.
* **Session manager duplication:** `StartChat/GetActiveChat2/EndChat` in
  `manager.go` vs `SetActiveChat/GetActiveChat/DeleteActiveChat` in `chat.go`
  manage the *same* Redis keys with two different value formats (plain partner
  ID vs JSON). `GetActiveChat` even has a fallback parser for the legacy
  format. Pick one (the JSON `ChatSession`) and delete the other trio.
* **Mixed languages/encodings in code comments and two `utils` packages**
  (`internal/utils`, `pkg/utils`) — consolidate.
* `cmd/.DS_Store`, `._.DS_Store` committed — add `.gitignore`.

### 🔴 Security

1. **`RequireAdmin` existed but was never used**, and `utils.AdminID = "x"` was
   a placeholder. Now wired: admin IDs come from `ADMIN_IDS` env var
   (comma-separated Telegram IDs), and all admin commands sit behind the
   middleware which silently ignores non-admins.
2. **No authorization on profile actions:** any user can call
   `/user_<id>` for any ID. Blocked users can still view the blocker's profile.
   Consider checking the Block table in `ShowUserProfile`.
3. **Callback data is trusted:** `btnLike`, `btnDM`, `btnBlock` etc. parse the
   target ID straight from callback data. A user can forge callbacks against
   arbitrary IDs (e.g., like-bombing). Validate that the action makes sense
   (target exists, not self, not blocked).
4. **`AcceptChatHandler` doesn't verify a pending request existed** — a forged
   callback creates a chat session with any user. Store pending chat requests
   in Redis (like `contact_pending:`) and verify on accept.
5. **Rate limiter memory leak:** `lastRequest map[int64]time.Time` grows
   forever (one entry per user, never evicted). Add periodic pruning or use
   Redis with TTL.
6. **`generateUserID` modulo bias** (`b[i] % 62`) — cosmetic, but
   `crypto/rand` + `big.Int` or rejection sampling is the right way.
7. **HTML parse mode with user content:** profile text is built with
   user-supplied name/city and sent with `ModeHTML` — escape user fields
   (`html.EscapeString`) or you get broken rendering / tag injection.

### 🟠 Performance & scalability

1. **Match worker is O(n²) per 2-second tick** over *all* queue entries, and
   every tick does `SCAN` + `LRANGE` of every queue. Fine for hundreds of
   concurrent searchers; will degrade beyond that. The gender/filter queues
   are already separate Redis lists — match heads of complementary lists
   (`LPOP` from `male` queue vs `female` queue) for O(1) matching.
2. **Long polling + single process:** state is in Redis/Postgres (good — the
   process is mostly stateless), but Telegram long-polling means exactly one
   instance. For horizontal scale, switch to webhooks behind a load balancer.
3. **`GetOrCreate` is called at the top of nearly every handler** → one
   Postgres `SELECT` per update. There's a `cached_repository.go` with a Redis
   cache — it's not used by `UserService`. Wire it in (with invalidation on
   `Update`) and most handlers stop touching Postgres entirely.
4. **N+1-ish work in `ProfileActionKeyboard`:** `IsBlocked` + `IsContact` are
   two separate queries per profile render; combine into one query or cache.
5. **`GetAllLastSeen`/`SyncLastSeenStream`** use SCAN properly — good. But the
   sync groups by exact unix timestamp, producing one `UPDATE … CASE` per
   distinct second. Group by nothing and use a single bulk update with VALUES.
6. **Connection pool:** 25 max conns is sane. `AutoMigrate` at boot is OK for
   now; move to versioned migrations (golang-migrate / atlas) before prod.
7. **Broadcast throttling:** the new `/broadcast` sleeps 40 ms between sends
   (~25 msg/s, under the Bot API's ~30/s global limit) and runs in a goroutine
   so the poller isn't blocked.

### 🟡 Correctness bugs found while reading

* `chatQueueTTL` was **1 minute** while the queue was supposed to hold users
  "until matched" — entries silently vanished after 60s with **no refund and
  no notification**. This is exactly the bug your new requirement #3 fixes:
  now there's an explicit 2-minute timeout with refund + "Search Again".
* `EditProfileHandler` called `h.bot.Edit(msg, kb)` with only a keyboard — if
  the profile message is a photo this fails silently and returns the error to
  telebot (user sees nothing).
* `ChatGameRequestCallback`/`Accept`/`Reject` dereference `cs` without nil
  check → panic (caught by Recovery middleware, but still a 500 per click) if
  the chat ended between sending and clicking.
* `repeatGameHandler`: `msg1, _ := h.bot.Send(...)` then `msg1.ID` — nil
  dereference if Send fails (user blocked the bot).
* `formatUserRow` prints `u.ID` (string short-ID) — fine — but
  `ConfirmEndChatHandler` prints `/user_%d` with the **TelegramID**, leaking
  the partner's numeric Telegram ID and producing a link that `TextHandler`
  can't resolve (it looks up by short ID). Use `partner.ID` in both places.
* `pkg/middleware/auth.go` took `*map[int64]bool` — maps are reference types;
  pointer-to-map is a smell (kept signature for compatibility, added nil
  checks).

---

## 2. ✅ Inline-button responses now edit instead of send

A single helper in `internal/bot/handlers.go` implements the rule everywhere:

```go
func editOrSend(c tele.Context, what interface{}, opts ...interface{}) error
```

* If the update is a **callback** (inline button click), it **edits** the
  originating message.
* It degrades to `Send` only when editing is impossible by Bot API rules:
  reply keyboards (`MainMenuKeyboard`, `ActiveChatKeyboard`, …) can only be
  delivered with a new message, photo captions vs text, deleted/old messages.
* All `c.Send(...)` calls in callback-reachable handlers across
  `01_start.go`–`10_game.go` were converted to `editOrSend(...)`.
* The match worker got the same treatment: when a match is found, the
  "Searching…" message is **edited in place** to "👀 پیدا شد!…" and only the
  reply keyboard ships as a new message; on timeout the searching message is
  edited into the "no user found" notice.

## 3. ✅ Search system: 2-minute auto search + «Search Again»

Implemented in `internal/bot/08_chat.go`, `match_worker.go`, `keyboards.go`,
`internal/session/chat.go`:

* **Manual cancel removed**: `WaitingKeyboard` (the «❌ لغو جستجو» button),
  `btnCancelQueue`, and `CancelQueueHandler` are gone; the searching message
  now carries no buttons.
* **`session.SearchTimeout = 2 * time.Minute`** is the single source of truth.
  `chatQueueTTL` was raised from 1 min to 5 min so Redis no longer silently
  evicts entries before the worker can time them out (this was a refund-eating
  bug).
* The match worker's tick now calls **`expireStaleEntries`**: any entry older
  than 2 minutes is removed from the queue, its **coins are refunded**
  (`search_timeout_refund`), and its searching message is **edited** into
  `messages.QueueNoMatch` with a **«🔁 جستجوی مجدد» inline button**.
* The button's callback data carries the user's **previous filter**
  (`random`, `male`, `female`, `nearby`, `nearby_male`, `nearby_female`), so
  `SearchAgainHandler` re-joins the queue with **the exact same filters**,
  re-checking and re-charging the correct coin cost via the new
  `costForFilter` helper (shared with the normal join path).

## 4. ✅ `pkg/messages` — centralized bot messages

* New package **`pkg/messages`** holds *every* user-facing string: onboarding,
  invites, coins, GPS, profile editing, search, queue/chat, DM, all five
  games, help pages, rules, and the new admin texts — grouped and commented.
* `internal/utils/messages.go` was **deleted**; the ~40 scattered hardcoded
  literals in `internal/bot/*` (`"❌ خطا"`, `"✅ پیام ارسال شد."`, game
  prompts, edit prompts, …) were migrated to named constants.
* All importers (`internal/bot`, `internal/game/*`, `internal/service`) now
  reference `GapGame/pkg/messages`. Changing any text is a one-line edit in
  one file.

## 5. ✅ Admin-only commands

New file `internal/bot/11_admin.go` (+ repo/service methods appended to the
**existing** `repository.go` / `service/user.go`, per your constraint), wired
behind `middleware.RequireAdmin`:

| Command | Behavior |
|---|---|
| `/admin` | Lists admin commands |
| `/broadcast <text>` | Sends to **all non-banned users**, throttled at ~25 msg/s in a background goroutine, reports success/failure counts back to the admin |
| `/give_coins_all <n>` | Single atomic SQL `UPDATE … coins = coins + n` for **all users** |
| `/give_coins_poor <n>` | Same, but only `WHERE coins < 2` |
| `/ban <telegram-id | /user_xxx>` | Sets `users.banned = true`, notifies the user |
| `/unban <telegram-id | /user_xxx>` | Clears the flag, notifies the user |

Supporting changes:

* `user.User` gained `Banned bool` (indexed, auto-migrated).
* **Ban guard middleware** (`banGuardMiddleware`) blocks every interaction
  from banned users with a support notice; admins are exempt.
* Admin IDs are configured via **`ADMIN_IDS`** env var
  (e.g. `ADMIN_IDS=123456789,987654321`), parsed in `config.Load()`.
* `ResolveTelegramID` accepts both numeric Telegram IDs and the bot's
  `/user_xxx` short IDs, so admins can ban directly from a report.

---

## Recommended next steps (in priority order)

1. Make all coin mutations atomic (`UPDATE coins = coins ± ?` with guard).
2. Delete dead code: `internal/matching/`, `bot/search_service.go`,
   `pkg/cleanup/service_cleaner.go`, `storage/sqlite.go`, `storage/connection_pool.go`,
   and the legacy active-chat trio in `session/manager.go`.
3. Validate pending chat requests on accept; authorize callback targets.
4. Escape user-supplied fields in HTML-mode messages.
5. Add `.gitignore` (logs, `.DS_Store`, `.env`), reduce GORM log level,
   move to versioned migrations.
6. Add tests around the queue/timeout/refund path and the admin commands.
7. Prune the rate-limit map periodically.
