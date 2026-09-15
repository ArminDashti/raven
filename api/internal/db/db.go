package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ArminDashti/raven-api/internal/auth"
	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return sqlDB, nil
}

func Migrate(sqlDB *sql.DB) error {
	_, err := sqlDB.Exec(`
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('tester','developer','manager','admin')),
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bug_reports (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  url TEXT NOT NULL,
  description TEXT NOT NULL,
  page_parameters TEXT NOT NULL,
  tested_with_user TEXT NOT NULL DEFAULT '',
  reporter_user_id INTEGER NOT NULL REFERENCES users(id),
  fixer_user_id INTEGER REFERENCES users(id),
  status TEXT NOT NULL CHECK(status IN ('reported','fixed','waiting_manager','done')),
  priority TEXT NOT NULL DEFAULT 'normal' CHECK(priority IN ('low','normal','high')),
  cycle INTEGER NOT NULL DEFAULT 1,
  rejection_reason TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bug_attachments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  bug_report_id INTEGER NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  original_filename TEXT NOT NULL,
  stored_filename TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bug_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  bug_report_id INTEGER NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  actor_user_id INTEGER NOT NULL REFERENCES users(id),
  url TEXT NOT NULL,
  status TEXT NOT NULL,
  rejection_reason TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notifications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id),
  bug_report_id INTEGER NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  is_read INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS push_subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  endpoint TEXT NOT NULL UNIQUE,
  p256dh TEXT NOT NULL,
  auth TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS team_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  bug_report_id INTEGER NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  author_user_id INTEGER NOT NULL REFERENCES users(id),
  body TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_team_messages_bug_created ON team_messages(bug_report_id, created_at);
`)
	if err != nil {
		return err
	}
	if err := migrateUsersAdminRole(sqlDB); err != nil {
		return err
	}
	if err := migrateLegacyStatuses(sqlDB); err != nil {
		return err
	}
	if err := migrateSkippedTesterWaitingManager(sqlDB); err != nil {
		return err
	}
	if err := migrateManagerReportedFixedToWaitingManager(sqlDB); err != nil {
		return err
	}
	if err := migrateTestedWithUser(sqlDB); err != nil {
		return err
	}
	if err := migrateBugCycle(sqlDB); err != nil {
		return err
	}
	if err := migrateBugPriority(sqlDB); err != nil {
		return err
	}
	if err := migrateTeamMessagesBugID(sqlDB); err != nil {
		return err
	}
	if err := migrateCapitalizeUsernames(sqlDB); err != nil {
		return err
	}
	if err := migrateUserNames(sqlDB); err != nil {
		return err
	}
	if err := migrateUserGender(sqlDB); err != nil {
		return err
	}
	if err := migrateNotificationMessage(sqlDB); err != nil {
		return err
	}
	return migrateUserAvatar(sqlDB)
}

// migrateUserAvatar adds avatar_filename and avatar_style (additive only).
// Never rebuilds users; never changes existing row values.
func migrateUserAvatar(sqlDB *sql.DB) error {
	for _, col := range []string{"avatar_filename", "avatar_style"} {
		var n int
		if err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM pragma_table_info('users') WHERE name = ?`, col).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if _, err := sqlDB.Exec(`ALTER TABLE users ADD COLUMN ` + col + ` TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add users.%s: %w", col, err)
		}
	}
	return nil
}

// migrateUserGender adds gender (man|woman|empty) and backfills known seed users.
func migrateUserGender(sqlDB *sql.DB) error {
	var n int
	if err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM pragma_table_info('users') WHERE name = 'gender'`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		if _, err := sqlDB.Exec(`ALTER TABLE users ADD COLUMN gender TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add users.gender: %w", err)
		}
	}
	women := []string{"Manager", "Tester2"}
	for _, u := range women {
		if _, err := sqlDB.Exec(`UPDATE users SET gender = 'woman' WHERE username = ? AND (gender = '' OR gender IS NULL)`, u); err != nil {
			return err
		}
	}
	men := []string{"Admin", "Developer", "Tester1"}
	for _, u := range men {
		if _, err := sqlDB.Exec(`UPDATE users SET gender = 'man' WHERE username = ? AND (gender = '' OR gender IS NULL)`, u); err != nil {
			return err
		}
	}
	return nil
}

// migrateNotificationMessage stores the human-readable message shown in the inbox / push body.
func migrateNotificationMessage(sqlDB *sql.DB) error {
	var n int
	if err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM pragma_table_info('notifications') WHERE name = 'message'`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if _, err := sqlDB.Exec(`ALTER TABLE notifications ADD COLUMN message TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add notifications.message: %w", err)
	}
	_, err := sqlDB.Exec(`
UPDATE notifications
SET message = COALESCE((
  SELECT CASE
    WHEN notifications.kind = 'rejected_for_dev' AND IFNULL(b.rejection_reason,'') != '' THEN b.rejection_reason
    ELSE b.description
  END
  FROM bug_reports b WHERE b.id = notifications.bug_report_id
), '')
WHERE message = ''`)
	return err
}

// migrateUserNames adds first_name/last_name (additive) and backfills known users.
// Never rebuilds users; never changes username, password_hash, or role.
func migrateUserNames(sqlDB *sql.DB) error {
	for _, col := range []string{"first_name", "last_name"} {
		var n int
		if err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM pragma_table_info('users') WHERE name = ?`, col).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if _, err := sqlDB.Exec(`ALTER TABLE users ADD COLUMN ` + col + ` TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add users.%s: %w", col, err)
		}
	}
	return backfillUserNames(sqlDB)
}

func backfillUserNames(sqlDB *sql.DB) error {
	type namePair struct {
		Username  string
		FirstName string
		LastName  string
	}
	names := []namePair{
		{"Admin", "Admin", ""},
		{"Developer", "Alex", "Dev"},
		{"Manager", "Sam", "Manager"},
		{"Tester1", "Casey", "Tester"},
		{"Tester2", "Jordan", "Tester"},
	}
	for _, u := range names {
		if _, err := sqlDB.Exec(
			`UPDATE users SET first_name = ?, last_name = ? WHERE username = ?`,
			u.FirstName, u.LastName, u.Username,
		); err != nil {
			return fmt.Errorf("backfill names for %q: %w", u.Username, err)
		}
	}
	return nil
}

// migrateTeamMessagesBugID adds bug_report_id for DBs that created the global chat table earlier.
func migrateTeamMessagesBugID(sqlDB *sql.DB) error {
	var n int
	if err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM pragma_table_info('team_messages') WHERE name = 'bug_report_id'`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if _, err := sqlDB.Exec(`ALTER TABLE team_messages ADD COLUMN bug_report_id INTEGER NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	_, err := sqlDB.Exec(`CREATE INDEX IF NOT EXISTS idx_team_messages_bug_created ON team_messages(bug_report_id, created_at)`)
	return err
}

// migrateSkippedTesterWaitingManager moves tester-reported bugs that jumped straight to manager
// (Dev stored waiting_manager while history only has fixed) back to fixed for Tester review.
// Manager-reported bugs legitimately skip Tester (Dev ↔ Manager), so they are excluded.
// UPDATE-only: never deletes bugs, history, notifications, or related rows.
func migrateSkippedTesterWaitingManager(sqlDB *sql.DB) error {
	_, err := sqlDB.Exec(`
UPDATE bug_reports
SET status = 'fixed'
WHERE status = 'waiting_manager'
  AND reporter_user_id NOT IN (SELECT id FROM users WHERE role = 'manager')
  AND id NOT IN (
    SELECT DISTINCT bug_report_id FROM bug_history WHERE status = 'waiting_manager'
  )`)
	return err
}

// migrateManagerReportedFixedToWaitingManager advances Manager-reported bugs that are still on
// "fixed" (Tester step) straight to waiting_manager, and records waiting_manager history when missing.
func migrateManagerReportedFixedToWaitingManager(sqlDB *sql.DB) error {
	rows, err := sqlDB.Query(`
SELECT b.id, b.url, COALESCE(b.fixer_user_id, b.reporter_user_id)
FROM bug_reports b
JOIN users u ON u.id = b.reporter_user_id
WHERE u.role = 'manager'
  AND (
    b.status = 'fixed'
    OR (
      b.status = 'waiting_manager'
      AND b.id NOT IN (
        SELECT DISTINCT bug_report_id FROM bug_history WHERE status = 'waiting_manager'
      )
    )
  )`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		id      int64
		url     string
		actorID int64
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.url, &r.actorID); err != nil {
			return err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, r := range list {
		if _, err := sqlDB.Exec(`
UPDATE bug_reports SET status = 'waiting_manager', updated_at = ? WHERE id = ?`, now, r.id); err != nil {
			return err
		}
		var n int
		if err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM bug_history WHERE bug_report_id = ? AND status = 'waiting_manager'`, r.id).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if _, err := sqlDB.Exec(`
INSERT INTO bug_history (bug_report_id, actor_user_id, url, status, rejection_reason, created_at)
VALUES (?, ?, ?, 'waiting_manager', NULL, ?)`, r.id, r.actorID, r.url, now); err != nil {
			return err
		}
	}
	return nil
}

func migrateTestedWithUser(sqlDB *sql.DB) error {
	var n int
	err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM pragma_table_info('bug_reports') WHERE name = 'tested_with_user'`).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = sqlDB.Exec(`ALTER TABLE bug_reports ADD COLUMN tested_with_user TEXT NOT NULL DEFAULT ''`)
	return err
}

func migrateBugPriority(sqlDB *sql.DB) error {
	var n int
	err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM pragma_table_info('bug_reports') WHERE name = 'priority'`).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = sqlDB.Exec(`ALTER TABLE bug_reports ADD COLUMN priority TEXT NOT NULL DEFAULT 'normal'`)
	return err
}

// migrateBugCycle adds cycle (tester→dev rounds) and syncs from rejection history.
// Cycle 1 = first report; each rejection that returns the bug to Dev adds one cycle.
func migrateBugCycle(sqlDB *sql.DB) error {
	var n int
	err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM pragma_table_info('bug_reports') WHERE name = 'cycle'`).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := sqlDB.Exec(`ALTER TABLE bug_reports ADD COLUMN cycle INTEGER NOT NULL DEFAULT 1`); err != nil {
			return err
		}
	}
	_, err = sqlDB.Exec(`
UPDATE bug_reports
SET cycle = 1 + (
  SELECT COUNT(1) FROM bug_history h
  WHERE h.bug_report_id = bug_reports.id AND h.status = 'rejected'
)`)
	return err
}

func migrateCapitalizeUsernames(sqlDB *sql.DB) error {
	rows, err := sqlDB.Query(`SELECT id, username FROM users`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		id       int64
		username string
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.username); err != nil {
			return err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, r := range list {
		next := capitalizePersonName(r.username)
		if next == r.username || next == "" {
			continue
		}
		if _, err := sqlDB.Exec(`UPDATE users SET username = ? WHERE id = ?`, next, r.id); err != nil {
			return fmt.Errorf("capitalize username %q: %w", r.username, err)
		}
	}
	return nil
}

func capitalizePersonName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == ' ' || r == '.' || r == '_' || r == '-'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		runes := []rune(p)
		if len(runes) == 0 {
			continue
		}
		first := strings.ToUpper(string(runes[0]))
		rest := ""
		if len(runes) > 1 {
			rest = strings.ToLower(string(runes[1:]))
		}
		out = append(out, first+rest)
	}
	return strings.Join(out, " ")
}

func migrateUsersAdminRole(sqlDB *sql.DB) error {
	var schemaSQL string
	err := sqlDB.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='users'`).Scan(&schemaSQL)
	if err != nil {
		return err
	}
	if strings.Contains(schemaSQL, "'admin'") {
		return nil
	}

	if _, err := sqlDB.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	defer func() { _, _ = sqlDB.Exec(`PRAGMA foreign_keys=ON`) }()

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DROP TABLE IF EXISTS users_new`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
CREATE TABLE users_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('tester','developer','manager','admin')),
  created_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("create users_new: %w", err)
	}
	if _, err := tx.Exec(`
INSERT INTO users_new (id, username, password_hash, role, created_at)
SELECT id, username, password_hash, role, created_at FROM users`); err != nil {
		return fmt.Errorf("copy users: %w", err)
	}
	if _, err := tx.Exec(`DROP TABLE users`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE users_new RENAME TO users`); err != nil {
		return err
	}
	return tx.Commit()
}

func migrateLegacyStatuses(sqlDB *sql.DB) error {
	var needsRebuild int
	err := sqlDB.QueryRow(`
SELECT COUNT(1) FROM bug_reports
WHERE status IN ('pending','changed_by_dev','accepted','rejected')`).Scan(&needsRebuild)
	if err != nil {
		// Table may be empty / new — check sql schema via pragma
		needsRebuild = 0
	}

	var legacySchema int
	_ = sqlDB.QueryRow(`
SELECT COUNT(1) FROM sqlite_master
WHERE type='table' AND name='bug_reports'
  AND sql LIKE '%pending%'`).Scan(&legacySchema)

	if needsRebuild == 0 && legacySchema == 0 {
		// Still remap history labels if any leftover
		_, _ = sqlDB.Exec(`UPDATE bug_history SET status = 'reported' WHERE status = 'pending'`)
		_, _ = sqlDB.Exec(`UPDATE bug_history SET status = 'fixed' WHERE status = 'changed_by_dev'`)
		_, _ = sqlDB.Exec(`UPDATE bug_history SET status = 'done' WHERE status IN ('accepted','accept')`)
		_, _ = sqlDB.Exec(`UPDATE bug_history SET status = 'rejected' WHERE status = 'reject'`)
		return nil
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DROP TABLE IF EXISTS bug_reports_new`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
CREATE TABLE bug_reports_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  url TEXT NOT NULL,
  description TEXT NOT NULL,
  page_parameters TEXT NOT NULL,
  tested_with_user TEXT NOT NULL DEFAULT '',
  reporter_user_id INTEGER NOT NULL REFERENCES users(id),
  fixer_user_id INTEGER REFERENCES users(id),
  status TEXT NOT NULL CHECK(status IN ('reported','fixed','waiting_manager','done')),
  cycle INTEGER NOT NULL DEFAULT 1,
  rejection_reason TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("create bug_reports_new: %w", err)
	}

	if _, err := tx.Exec(`
INSERT INTO bug_reports_new (
  id, url, description, page_parameters, tested_with_user, reporter_user_id, fixer_user_id,
  status, cycle, rejection_reason, created_at, updated_at
)
SELECT
  id, url, description, page_parameters, '', reporter_user_id, fixer_user_id,
  CASE status
    WHEN 'pending' THEN 'reported'
    WHEN 'changed_by_dev' THEN 'fixed'
    WHEN 'accepted' THEN 'done'
    WHEN 'rejected' THEN 'reported'
    WHEN 'reported' THEN 'reported'
    WHEN 'fixed' THEN 'fixed'
    WHEN 'waiting_manager' THEN 'waiting_manager'
    WHEN 'done' THEN 'done'
    ELSE 'reported'
  END,
  1,
  rejection_reason, created_at, updated_at
FROM bug_reports`); err != nil {
		return fmt.Errorf("copy bug_reports: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE bug_reports`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE bug_reports_new RENAME TO bug_reports`); err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE bug_history SET status = 'reported' WHERE status = 'pending'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE bug_history SET status = 'fixed' WHERE status = 'changed_by_dev'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE bug_history SET status = 'done' WHERE status IN ('accepted','accept')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE bug_history SET status = 'rejected' WHERE status = 'reject'`); err != nil {
		return err
	}

	return tx.Commit()
}

func SeedUsers(sqlDB *sql.DB) error {
	type seedUser struct {
		Username  string
		Role      string
		FirstName string
		LastName  string
	}
	users := []seedUser{
		{"Admin", "admin", "Admin", ""},
		{"Developer", "developer", "Alex", "Dev"},
		{"Manager", "manager", "Sam", "Manager"},
		{"Tester1", "tester", "Casey", "Tester"},
		{"Tester2", "tester", "Jordan", "Tester"},
	}
	hash, err := auth.HashPassword("123")
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, u := range users {
		var exists int
		if err := sqlDB.QueryRow(`SELECT COUNT(1) FROM users WHERE username = ?`, u.Username).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			if _, err := sqlDB.Exec(
				`UPDATE users SET first_name = ?, last_name = ? WHERE username = ?`,
				u.FirstName, u.LastName, u.Username,
			); err != nil {
				return fmt.Errorf("seed names %s: %w", u.Username, err)
			}
			continue
		}
		if _, err := sqlDB.Exec(
			`INSERT INTO users (username, password_hash, role, first_name, last_name, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			u.Username, hash, u.Role, u.FirstName, u.LastName, now,
		); err != nil {
			return fmt.Errorf("seed %s: %w", u.Username, err)
		}
	}
	return nil
}
