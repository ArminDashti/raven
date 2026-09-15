package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ArminDashti/raven-api/internal/auth"
	"github.com/ArminDashti/raven-api/internal/middleware"
	"github.com/ArminDashti/raven-api/internal/push"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	DB        *sql.DB
	JWTSecret string
	UploadDir string
	Push      *push.Sender
}

func New(db *sql.DB, jwtSecret, uploadDir string, pushSender *push.Sender) *Handler {
	return &Handler{DB: db, JWTSecret: jwtSecret, UploadDir: uploadDir, Push: pushSender}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}
	var id int64
	var hash, role, storedUsername, firstName, lastName, gender, avatarFilename, avatarStyle string
	err := h.DB.QueryRow(
		`SELECT id, password_hash, role, username, COALESCE(first_name,''), COALESCE(last_name,''), COALESCE(gender,''), COALESCE(avatar_filename,''), COALESCE(avatar_style,'') FROM users WHERE lower(username) = lower(?)`,
		strings.TrimSpace(req.Username),
	).Scan(&id, &hash, &role, &storedUsername, &firstName, &lastName, &gender, &avatarFilename, &avatarStyle)
	if err != nil || !auth.CheckPassword(hash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token, err := auth.IssueToken(h.JWTSecret, id, storedUsername, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  h.userPublicJSON(id, storedUsername, role, firstName, lastName, gender, "", avatarFilename, avatarStyle),
	})
}

func (h *Handler) userPublicJSON(id int64, username, role, firstName, lastName, gender, createdAt, avatarFilename, avatarStyle string) gin.H {
	out := gin.H{
		"id":            id,
		"username":      username,
		"role":          role,
		"first_name":    firstName,
		"last_name":     lastName,
		"gender":        gender,
		"avatar_style":  avatarStyle,
		"has_avatar":    avatarFilename != "",
		"avatar_url":    "",
	}
	if createdAt != "" {
		out["created_at"] = createdAt
	}
	if avatarFilename != "" {
		out["avatar_url"] = fmt.Sprintf("/api/v1/users/%d/avatar", id)
	}
	return out
}

func (h *Handler) Me(c *gin.Context) {
	userID := c.GetInt64(middleware.ContextUserID)
	var username, role, createdAt, firstName, lastName, gender, avatarFilename, avatarStyle string
	err := h.DB.QueryRow(
		`SELECT username, role, created_at, COALESCE(first_name,''), COALESCE(last_name,''), COALESCE(gender,''), COALESCE(avatar_filename,''), COALESCE(avatar_style,'') FROM users WHERE id = ?`,
		userID,
	).Scan(&username, &role, &createdAt, &firstName, &lastName, &gender, &avatarFilename, &avatarStyle)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, h.userPublicJSON(userID, username, role, firstName, lastName, gender, createdAt, avatarFilename, avatarStyle))
}

func (h *Handler) UpdateMe(c *gin.Context) {
	userID := c.GetInt64(middleware.ContextUserID)
	var req struct {
		Gender      *string `json:"gender"`
		AvatarStyle *string `json:"avatar_style"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Gender == nil && req.AvatarStyle == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gender or avatar_style is required"})
		return
	}
	if req.Gender != nil {
		gender := strings.TrimSpace(strings.ToLower(*req.Gender))
		switch gender {
		case "", "man", "woman":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "gender must be man, woman, or empty"})
			return
		}
		if _, err := h.DB.Exec(`UPDATE users SET gender = ? WHERE id = ?`, gender, userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
			return
		}
	}
	if req.AvatarStyle != nil {
		style := strings.TrimSpace(strings.ToLower(*req.AvatarStyle))
		switch style {
		case "", "man", "woman", "nerd", "robot", "fox", "abstract":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid avatar_style"})
			return
		}
		if _, err := h.DB.Exec(`UPDATE users SET avatar_style = ? WHERE id = ?`, style, userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
			return
		}
	}
	h.Me(c)
}

func (h *Handler) ListUsers(c *gin.Context) {
	rows, err := h.DB.Query(
		`SELECT id, username, role, COALESCE(first_name,''), COALESCE(last_name,''), COALESCE(gender,''), COALESCE(avatar_filename,''), COALESCE(avatar_style,'') FROM users ORDER BY username`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id int64
		var username, role, firstName, lastName, gender, avatarFilename, avatarStyle string
		if err := rows.Scan(&id, &username, &role, &firstName, &lastName, &gender, &avatarFilename, &avatarStyle); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, h.userPublicJSON(id, username, role, firstName, lastName, gender, "", avatarFilename, avatarStyle))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) GetUserByUsername(c *gin.Context) {
	// Route param is :id (shared Gin wildcard with /users/:id/avatar); value is username.
	username := strings.TrimSpace(c.Param("id"))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid username"})
		return
	}
	var id int64
	var role, createdAt, firstName, lastName, gender, avatarFilename, avatarStyle, storedUsername string
	err := h.DB.QueryRow(
		`SELECT id, username, role, created_at, COALESCE(first_name,''), COALESCE(last_name,''), COALESCE(gender,''), COALESCE(avatar_filename,''), COALESCE(avatar_style,'') FROM users WHERE lower(username) = lower(?)`,
		username,
	).Scan(&id, &storedUsername, &role, &createdAt, &firstName, &lastName, &gender, &avatarFilename, &avatarStyle)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, h.userPublicJSON(id, storedUsername, role, firstName, lastName, gender, createdAt, avatarFilename, avatarStyle))
}

const maxAvatarBytes = 2 * 1024 * 1024

func (h *Handler) UploadAvatar(c *gin.Context) {
	userID := c.GetInt64(middleware.ContextUserID)
	fh, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar file is required"})
		return
	}
	if fh.Size > maxAvatarBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar exceeds 2MB limit"})
		return
	}
	ct := strings.ToLower(fh.Header.Get("Content-Type"))
	if !strings.HasPrefix(ct, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar must be an image"})
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		ext = ".png"
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		return
	}
	stored := fmt.Sprintf("avatar_%d_%s%s", userID, hex.EncodeToString(buf[:]), ext)
	dst := filepath.Join(h.UploadDir, stored)
	if err := c.SaveUploadedFile(fh, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	var oldName string
	_ = h.DB.QueryRow(`SELECT COALESCE(avatar_filename,'') FROM users WHERE id = ?`, userID).Scan(&oldName)
	if _, err := h.DB.Exec(`UPDATE users SET avatar_filename = ? WHERE id = ?`, stored, userID); err != nil {
		_ = os.Remove(dst)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if oldName != "" && oldName != stored {
		_ = os.Remove(filepath.Join(h.UploadDir, oldName))
	}
	h.Me(c)
}

func (h *Handler) DownloadUserAvatar(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var stored string
	if err := h.DB.QueryRow(`SELECT COALESCE(avatar_filename,'') FROM users WHERE id = ?`, userID).Scan(&stored); err != nil || stored == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		return
	}
	path := filepath.Join(h.UploadDir, stored)
	c.File(path)
}

const teamChatMaxBody = 2000

func (h *Handler) teamChatBugID(c *gin.Context) (int64, bool) {
	bugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || bugID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bug id"})
		return 0, false
	}
	var n int
	if err := h.DB.QueryRow(`SELECT COUNT(1) FROM bug_reports WHERE id = ?`, bugID).Scan(&n); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return 0, false
	}
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bug not found"})
		return 0, false
	}
	return bugID, true
}

func (h *Handler) ListTeamMessages(c *gin.Context) {
	bugID, ok := h.teamChatBugID(c)
	if !ok {
		return
	}
	rows, err := h.DB.Query(`
SELECT id, body, created_at, author_user_id, author_username, author_role FROM (
  SELECT m.id, m.body, m.created_at, m.author_user_id,
         u.username AS author_username, u.role AS author_role
  FROM team_messages m
  JOIN users u ON u.id = m.author_user_id
  WHERE m.bug_report_id = ?
  ORDER BY m.id DESC
  LIMIT 200
) t
ORDER BY id ASC`, bugID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, authorID int64
		var body, createdAt, username, role string
		if err := rows.Scan(&id, &body, &createdAt, &authorID, &username, &role); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, gin.H{
			"id":              id,
			"body":            body,
			"created_at":      createdAt,
			"author_user_id":  authorID,
			"author_username": username,
			"author_role":     role,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) CreateTeamMessage(c *gin.Context) {
	bugID, ok := h.teamChatBugID(c)
	if !ok {
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}
	if len([]rune(body)) > teamChatMaxBody {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is too long"})
		return
	}
	authorID := c.GetInt64(middleware.ContextUserID)
	now := time.Now().UTC().Format(time.RFC3339)

	var url string
	var reporterID int64
	var fixerID sql.NullInt64
	if err := h.DB.QueryRow(`SELECT url, reporter_user_id, fixer_user_id FROM bug_reports WHERE id = ?`, bugID).
		Scan(&url, &reporterID, &fixerID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bug not found"})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
INSERT INTO team_messages (bug_report_id, author_user_id, body, created_at)
VALUES (?, ?, ?, ?)`, bugID, authorID, body, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}
	if _, err := tx.Exec(`UPDATE bug_reports SET updated_at = ? WHERE id = ?`, now, bugID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if _, err := tx.Exec(`
INSERT INTO bug_history (bug_report_id, actor_user_id, url, status, rejection_reason, created_at)
VALUES (?, ?, ?, 'team_message', NULL, ?)`, bugID, authorID, url, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "history failed"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}

	var username, role string
	if err := h.DB.QueryRow(`SELECT username, role FROM users WHERE id = ?`, authorID).
		Scan(&username, &role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "author lookup failed"})
		return
	}

	notifySet := map[int64]struct{}{}
	if reporterID > 0 && reporterID != authorID {
		notifySet[reporterID] = struct{}{}
	}
	if fixerID.Valid && fixerID.Int64 != authorID {
		notifySet[fixerID.Int64] = struct{}{}
	}
	if mgrIDs, err := userIDsByRole(h.DB, "manager"); err == nil {
		for _, uid := range mgrIDs {
			if uid != authorID {
				notifySet[uid] = struct{}{}
			}
		}
	}
	notifyIDs := make([]int64, 0, len(notifySet))
	for uid := range notifySet {
		notifyIDs = append(notifyIDs, uid)
		_, _ = h.DB.Exec(`
INSERT INTO notifications (user_id, bug_report_id, kind, is_read, created_at, message)
VALUES (?, ?, 'team_message', 0, ?, ?)`, uid, bugID, now, body)
	}
	if len(notifyIDs) > 0 {
		h.Push.NotifyUsers(notifyIDs, "New team message", body, fmt.Sprintf("/bugs/%d", bugID))
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              id,
		"body":            body,
		"created_at":      now,
		"author_user_id":  authorID,
		"author_username": username,
		"author_role":     role,
	})
}

func scanBug(row interface {
	Scan(dest ...any) error
}) (gin.H, error) {
	var id, reporterID, cycle int64
	var fixerID sql.NullInt64
	var url, description, pageParams, testedWith, status, priority, createdAt, updatedAt, reporterName, fixerName string
	var rejection sql.NullString
	var lastHistStatus, lastHistActor sql.NullString
	if err := row.Scan(&id, &url, &description, &pageParams, &testedWith, &status, &priority, &cycle, &rejection,
		&createdAt, &updatedAt, &reporterID, &reporterName, &fixerID, &fixerName,
		&lastHistStatus, &lastHistActor); err != nil {
		return nil, err
	}
	return gin.H{
		"id":                   id,
		"url":                  url,
		"description":          description,
		"page_parameters":       pageParams,
		"tested_with_user":     testedWith,
		"status":               status,
		"priority":             priority,
		"cycle":                cycle,
		"rejection_reason":     nullStr(rejection),
		"created_at":           createdAt,
		"updated_at":           updatedAt,
		"reporter_user_id":     reporterID,
		"reporter_username":    reporterName,
		"fixer_user_id":        nullInt(fixerID),
		"fixer_username":       fixerName,
		"last_history_status":  nullStr(lastHistStatus),
		"last_history_actor":   nullStr(lastHistActor),
	}, nil
}

const bugSelect = `
SELECT b.id, b.url, b.description, b.page_parameters, b.tested_with_user, b.status, b.priority, b.cycle, b.rejection_reason,
       b.created_at, b.updated_at,
       b.reporter_user_id, r.username,
       b.fixer_user_id, COALESCE(f.username, ''),
       (SELECT h.status FROM bug_history h WHERE h.bug_report_id = b.id ORDER BY h.id DESC LIMIT 1),
       (SELECT u.username FROM bug_history h JOIN users u ON u.id = h.actor_user_id WHERE h.bug_report_id = b.id ORDER BY h.id DESC LIMIT 1)
FROM bug_reports b
JOIN users r ON r.id = b.reporter_user_id
LEFT JOIN users f ON f.id = b.fixer_user_id`

func (h *Handler) ListBugReports(c *gin.Context) {
	rows, err := h.DB.Query(bugSelect + ` ORDER BY b.created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		item, err := scanBug(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, item)
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) GetBugReport(c *gin.Context) {
	bugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	row := h.DB.QueryRow(bugSelect+` WHERE b.id = ?`, bugID)
	item, err := scanBug(row)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bug not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) UpdateBugFields(c *gin.Context) {
	bugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Description    *string `json:"description"`
		PageParameters *string `json:"page_parameters"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Description == nil && req.PageParameters == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description or page_parameters required"})
		return
	}

	actorID := c.GetInt64(middleware.ContextUserID)
	var reporterID int64
	var url, curDesc, curParams string
	err = h.DB.QueryRow(`SELECT url, description, page_parameters, reporter_user_id FROM bug_reports WHERE id = ?`, bugID).
		Scan(&url, &curDesc, &curParams, &reporterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bug not found"})
		return
	}
	if actorID != reporterID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the bug reporter can edit description and page parameters"})
		return
	}

	newDesc := curDesc
	newParams := curParams
	var changed []string
	if req.Description != nil {
		newDesc = strings.TrimSpace(*req.Description)
		if newDesc == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "description cannot be empty"})
			return
		}
		if newDesc != curDesc {
			changed = append(changed, "description")
		}
	}
	if req.PageParameters != nil {
		newParams = strings.TrimSpace(*req.PageParameters)
		if newParams == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "page_parameters cannot be empty"})
			return
		}
		if newParams != curParams {
			changed = append(changed, "page parameters")
		}
	}
	if len(changed) == 0 {
		c.JSON(http.StatusOK, gin.H{"id": bugID, "ok": true})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	summary := "Updated " + strings.Join(changed, " and ")
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
UPDATE bug_reports SET description = ?, page_parameters = ?, updated_at = ? WHERE id = ?`,
		newDesc, newParams, now, bugID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if _, err := tx.Exec(`
INSERT INTO bug_history (bug_report_id, actor_user_id, url, status, rejection_reason, created_at)
VALUES (?, ?, ?, 'edit', ?, ?)`, bugID, actorID, url, summary, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "history failed"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": bugID, "ok": true})
}

func (h *Handler) DeleteBugReport(c *gin.Context) {
	if c.GetString(middleware.ContextRole) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can delete bug reports"})
		return
	}
	bugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	rows, err := h.DB.Query(`SELECT stored_filename FROM bug_attachments WHERE bug_report_id = ?`, bugID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query attachments failed"})
		return
	}
	var storedFiles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan attachments failed"})
			return
		}
		storedFiles = append(storedFiles, name)
	}
	rows.Close()

	res, err := h.DB.Exec(`DELETE FROM bug_reports WHERE id = ?`, bugID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bug not found"})
		return
	}

	for _, name := range storedFiles {
		_ = os.Remove(filepath.Join(h.UploadDir, name))
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "id": bugID})
}

func (h *Handler) ListAttachments(c *gin.Context) {
	bugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var exists int
	if err := h.DB.QueryRow(`SELECT COUNT(1) FROM bug_reports WHERE id = ?`, bugID).Scan(&exists); err != nil || exists == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bug not found"})
		return
	}
	rows, err := h.DB.Query(`
SELECT id, original_filename, content_type, size_bytes, created_at
FROM bug_attachments WHERE bug_report_id = ? ORDER BY id ASC`, bugID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, size int64
		var name, ct, createdAt string
		if err := rows.Scan(&id, &name, &ct, &size, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, gin.H{
			"id":                id,
			"original_filename": name,
			"content_type":      ct,
			"size_bytes":        size,
			"created_at":        createdAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

const (
	maxAttachmentFiles = 10
	maxAttachmentBytes = 10 * 1024 * 1024
)

func canReportRole(role string) bool {
	return role == "tester" || role == "developer" || role == "manager" || role == "admin"
}

func canDevelopRole(role string) bool {
	return role == "developer" || role == "manager" || role == "admin"
}

func canApproveRole(role string) bool {
	return role == "manager" || role == "admin"
}

func canTesterReviewRole(role string) bool {
	return role == "tester" || role == "admin"
}

func (h *Handler) CreateBugReport(c *gin.Context) {
	role := c.GetString(middleware.ContextRole)
	if !canReportRole(role) {
		c.JSON(http.StatusForbidden, gin.H{"error": "your role cannot create bug reports"})
		return
	}

	contentType := c.GetHeader("Content-Type")
	var attachmentFiles []*multipart.FileHeader
	if strings.HasPrefix(strings.ToLower(contentType), "multipart/") {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart upload (files may be too large)"})
			return
		}
		if form != nil {
			attachmentFiles = form.File["attachments"]
		}
		if len(attachmentFiles) > maxAttachmentFiles {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("at most %d attachment files allowed", maxAttachmentFiles)})
			return
		}
		for _, fh := range attachmentFiles {
			if fh.Size > maxAttachmentBytes {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file %q exceeds %d MB limit", fh.Filename, maxAttachmentBytes/(1024*1024))})
				return
			}
		}
	}

	url := strings.TrimSpace(c.PostForm("url"))
	description := strings.TrimSpace(c.PostForm("description"))
	pageParams := strings.TrimSpace(c.PostForm("page_parameters"))
	testedWith := strings.TrimSpace(c.PostForm("tested_with_user")) // optional; column kept for existing rows
	priority := strings.TrimSpace(c.PostForm("priority"))
	if priority == "" {
		priority = "normal"
	}
	if priority != "low" && priority != "normal" && priority != "high" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "priority must be low, normal, or high"})
		return
	}
	reporterIDStr := strings.TrimSpace(c.PostForm("reporter_user_id"))
	if url == "" || description == "" || pageParams == "" || reporterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url, description, page_parameters, and reporter_user_id are required"})
		return
	}
	reporterID, err := strconv.ParseInt(reporterIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reporter_user_id"})
		return
	}
	var reporterRole string
	if err := h.DB.QueryRow(`SELECT role FROM users WHERE id = ?`, reporterID).Scan(&reporterRole); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reporter not found"})
		return
	}
	if !canReportRole(reporterRole) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reporter role cannot report bugs"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	actorID := c.GetInt64(middleware.ContextUserID)

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
INSERT INTO bug_reports (url, description, page_parameters, tested_with_user, reporter_user_id, fixer_user_id, status, priority, cycle, rejection_reason, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, NULL, 'reported', ?, 1, NULL, ?, ?)`,
		url, description, pageParams, testedWith, reporterID, priority, now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}
	bugID, _ := res.LastInsertId()

	if _, err := tx.Exec(`
INSERT INTO bug_history (bug_report_id, actor_user_id, url, status, rejection_reason, created_at)
VALUES (?, ?, ?, 'reported', NULL, ?)`, bugID, actorID, url, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "history insert failed"})
		return
	}

	notifyFixerIDs, err := userIDsByRoles(tx, "developer", "manager")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "notify failed"})
		return
	}
	for _, uid := range notifyFixerIDs {
		if _, err := tx.Exec(`
INSERT INTO notifications (user_id, bug_report_id, kind, is_read, created_at, message)
VALUES (?, ?, 'reported_for_dev', 0, ?, ?)`, uid, bugID, now, description); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "notify insert failed"})
			return
		}
	}

	if len(attachmentFiles) > 0 {
		if err := os.MkdirAll(h.UploadDir, 0o755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "upload dir failed"})
			return
		}
		for _, fh := range attachmentFiles {
			src, err := fh.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "file open failed"})
				return
			}
			stored := randomName() + filepath.Ext(fh.Filename)
			dstPath := filepath.Join(h.UploadDir, stored)
			dst, err := os.Create(dstPath)
			if err != nil {
				_ = src.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "file create failed"})
				return
			}
			limited := io.LimitReader(src, maxAttachmentBytes+1)
			n, copyErr := io.Copy(dst, limited)
			_ = dst.Close()
			_ = src.Close()
			if copyErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "file save failed"})
				return
			}
			if n > maxAttachmentBytes {
				_ = os.Remove(dstPath)
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file %q exceeds %d MB limit", fh.Filename, maxAttachmentBytes/(1024*1024))})
				return
			}
			ct := fh.Header.Get("Content-Type")
			if ct == "" {
				ct = "application/octet-stream"
			}
			if _, err := tx.Exec(`
INSERT INTO bug_attachments (bug_report_id, original_filename, stored_filename, content_type, size_bytes, created_at)
VALUES (?, ?, ?, ?, ?, ?)`, bugID, fh.Filename, stored, ct, n, now); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "attachment insert failed"})
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	h.Push.NotifyUsers(notifyFixerIDs, "New bug reported", description, fmt.Sprintf("/bugs/%d", bugID))
	c.JSON(http.StatusCreated, gin.H{"id": bugID, "status": "reported"})
}

type statusRequest struct {
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason"`
}

func (h *Handler) UpdateBugStatus(c *gin.Context) {
	bugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req statusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	role := c.GetString(middleware.ContextRole)
	actorID := c.GetInt64(middleware.ContextUserID)

	var currentStatus, url, description string
	var reporterID int64
	var fixerID sql.NullInt64
	err = h.DB.QueryRow(`SELECT status, url, description, reporter_user_id, fixer_user_id FROM bug_reports WHERE id = ?`, bugID).
		Scan(&currentStatus, &url, &description, &reporterID, &fixerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bug not found"})
		return
	}
	var reporterRole string
	if err := h.DB.QueryRow(`SELECT role FROM users WHERE id = ?`, reporterID).Scan(&reporterRole); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reporter role lookup failed"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	historyStatus := req.Status
	storedStatus := req.Status
	var rejection *string
	var newFixer *int64
	var notifyIDs []int64
	var notifyKind, pushTitle, pushBody, notifyMessage string
	// When Manager reported: Dev fix also records a "fixed" history row, then lands on waiting_manager.
	recordFixedThenManager := false
	notifyMessage = description
	pushBody = description

	switch {
	// Dev: mark fixed → Tester review (or Manager finalize when Manager reported)
	case canDevelopRole(role) && currentStatus == "reported" && req.Status == "fixed":
		newFixer = &actorID
		if reporterRole == "manager" {
			// Manager-reported: Dev ↔ Manager only (skip Tester).
			recordFixedThenManager = true
			historyStatus = "waiting_manager"
			storedStatus = "waiting_manager"
			mgrIDs, err := userIDsByRole(h.DB, "manager")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "notify failed"})
				return
			}
			notifyIDs = mgrIDs
			notifyKind = "waiting_manager"
			pushTitle = "Bug awaiting manager finalize"
		} else {
			historyStatus = "fixed"
			storedStatus = "fixed"
			notifyIDs = notifyTestersForFixed(h.DB, reporterID)
			notifyKind = "fixed_for_tester"
			pushTitle = "Bug awaiting tester approval"
		}

	// Tester: approve → Manager finalize (only that bug's reporter, or admin)
	case canTesterReviewRole(role) && currentStatus == "fixed" && req.Status == "waiting_manager":
		if !isBugReporterOrAdmin(role, actorID, reporterID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the reporting tester can approve this bug"})
			return
		}
		historyStatus = "waiting_manager"
		storedStatus = "waiting_manager"
		mgrIDs, err := userIDsByRole(h.DB, "manager")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "notify failed"})
			return
		}
		notifyIDs = mgrIDs
		notifyKind = "waiting_manager"
		pushTitle = "Bug awaiting manager finalize"

	// Tester: reject → back to Dev (only that bug's reporter, or admin)
	case canTesterReviewRole(role) && currentStatus == "fixed" && req.Status == "rejected":
		if !isBugReporterOrAdmin(role, actorID, reporterID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the reporting tester can reject this bug"})
			return
		}
		reason := strings.TrimSpace(req.RejectionReason)
		if reason == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rejection_reason is required"})
			return
		}
		rejection = &reason
		historyStatus = "rejected"
		storedStatus = "reported"
		notifyIDs = notifyDevsForReject(h.DB, fixerID)
		notifyKind = "rejected_for_dev"
		pushTitle = "Bug rejected by tester"
		notifyMessage = reason
		pushBody = reason

	// Manager: accept → closed
	case canApproveRole(role) && currentStatus == "waiting_manager" && req.Status == "done":
		historyStatus = "done"
		storedStatus = "done"
		notifyIDs = []int64{reporterID}
		notifyKind = "done"
		pushTitle = "Bug done"

	// Manager: reject → back to Dev (then Tester or Manager cycle by reporter role)
	case canApproveRole(role) && currentStatus == "waiting_manager" && req.Status == "rejected":
		reason := strings.TrimSpace(req.RejectionReason)
		if reason == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rejection_reason is required"})
			return
		}
		rejection = &reason
		historyStatus = "rejected"
		storedStatus = "reported"
		notifyIDs = notifyDevsForReject(h.DB, fixerID)
		notifyKind = "rejected_for_dev"
		pushTitle = "Bug rejected by manager"
		notifyMessage = reason
		pushBody = reason

	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "status transition not allowed"})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	if newFixer != nil {
		_, err = tx.Exec(`UPDATE bug_reports SET status = ?, fixer_user_id = ?, updated_at = ?, rejection_reason = NULL WHERE id = ?`,
			storedStatus, *newFixer, now, bugID)
	} else if rejection != nil {
		_, err = tx.Exec(`UPDATE bug_reports SET status = ?, rejection_reason = ?, updated_at = ?, cycle = cycle + 1 WHERE id = ?`,
			storedStatus, *rejection, now, bugID)
	} else {
		_, err = tx.Exec(`UPDATE bug_reports SET status = ?, rejection_reason = NULL, updated_at = ? WHERE id = ?`,
			storedStatus, now, bugID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	var rejArg interface{}
	if rejection != nil {
		rejArg = *rejection
	}
	if recordFixedThenManager {
		if _, err := tx.Exec(`
INSERT INTO bug_history (bug_report_id, actor_user_id, url, status, rejection_reason, created_at)
VALUES (?, ?, ?, 'fixed', NULL, ?)`, bugID, actorID, url, now); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "history failed"})
			return
		}
	}
	if _, err := tx.Exec(`
INSERT INTO bug_history (bug_report_id, actor_user_id, url, status, rejection_reason, created_at)
VALUES (?, ?, ?, ?, ?, ?)`, bugID, actorID, url, historyStatus, rejArg, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "history failed"})
		return
	}

	for _, uid := range notifyIDs {
		if _, err := tx.Exec(`
INSERT INTO notifications (user_id, bug_report_id, kind, is_read, created_at, message)
VALUES (?, ?, ?, 0, ?, ?)`, uid, bugID, notifyKind, now, notifyMessage); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "notify insert failed"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	h.Push.NotifyUsers(notifyIDs, pushTitle, pushBody, fmt.Sprintf("/bugs/%d", bugID))
	c.JSON(http.StatusOK, gin.H{"id": bugID, "status": storedStatus})
}

func notifyDevsForReject(db *sql.DB, fixerID sql.NullInt64) []int64 {
	if fixerID.Valid {
		return []int64{fixerID.Int64}
	}
	ids, err := userIDsByRoles(db, "developer", "manager")
	if err != nil {
		return nil
	}
	return ids
}

func isBugReporterOrAdmin(role string, actorID, reporterID int64) bool {
	return role == "admin" || actorID == reporterID
}

// After Dev marks fixed, notify only the bug's reporter (its tester) — never every tester.
func notifyTestersForFixed(_ *sql.DB, reporterID int64) []int64 {
	return []int64{reporterID}
}

func userIDsByRoles(q querier, roles ...string) ([]int64, error) {
	seen := map[int64]struct{}{}
	var out []int64
	for _, role := range roles {
		ids, err := userIDsByRole(q, role)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out, nil
}

func (h *Handler) BugHistory(c *gin.Context) {
	bugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	rows, err := h.DB.Query(`
SELECT h.id, h.bug_report_id, h.url, h.status, h.rejection_reason, h.created_at,
       h.actor_user_id, u.username, r.username
FROM bug_history h
JOIN users u ON u.id = h.actor_user_id
JOIN bug_reports b ON b.id = h.bug_report_id
JOIN users r ON r.id = b.reporter_user_id
WHERE h.bug_report_id = ?
ORDER BY h.created_at ASC`, bugID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, bugReportID, actorID int64
		var url, status, createdAt, username, reporterName string
		var rejection sql.NullString
		if err := rows.Scan(&id, &bugReportID, &url, &status, &rejection, &createdAt, &actorID, &username, &reporterName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, gin.H{
			"id":                 id,
			"bug_report_id":      bugReportID,
			"url":                url,
			"status":             status,
			"rejection_reason":   nullStr(rejection),
			"created_at":         createdAt,
			"actor_user_id":      actorID,
			"actor_username":     username,
			"reporter_username":  reporterName,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) ListNotifications(c *gin.Context) {
	userID := c.GetInt64(middleware.ContextUserID)
	rows, err := h.DB.Query(`
SELECT n.id, n.bug_report_id, n.kind, n.is_read, n.created_at, COALESCE(n.message, ''),
       b.url, b.status, b.description, COALESCE(f.username, ''), COALESCE(r.username, ''), b.rejection_reason
FROM notifications n
JOIN bug_reports b ON b.id = n.bug_report_id
LEFT JOIN users f ON f.id = b.fixer_user_id
LEFT JOIN users r ON r.id = b.reporter_user_id
WHERE n.user_id = ?
ORDER BY n.is_read ASC, n.created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, bugID int64
		var isRead int
		var kind, createdAt, message, url, status, description, fixerName, reporterName string
		var rejection sql.NullString
		if err := rows.Scan(&id, &bugID, &kind, &isRead, &createdAt, &message, &url, &status, &description, &fixerName, &reporterName, &rejection); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		if strings.TrimSpace(message) == "" {
			if kind == "rejected_for_dev" && rejection.Valid && strings.TrimSpace(rejection.String) != "" {
				message = rejection.String
			} else {
				message = description
			}
		}
		out = append(out, gin.H{
			"id":                id,
			"bug_report_id":     bugID,
			"kind":              kind,
			"is_read":           isRead == 1,
			"created_at":        createdAt,
			"message":           message,
			"url":               url,
			"bug_status":        status,
			"fixer_username":    fixerName,
			"reporter_username": reporterName,
			"rejection_reason":  nullStr(rejection),
		})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	userID := c.GetInt64(middleware.ContextUserID)
	var count int
	if err := h.DB.QueryRow(`SELECT COUNT(1) FROM notifications WHERE user_id = ? AND is_read = 0`, userID).Scan(&count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (h *Handler) BugStats(c *gin.Context) {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		loc = time.FixedZone("IRST", 3*3600+30*60)
	}
	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	// Weeks start on Saturday (Sat=0 … Fri=6).
	daysSinceSat := (int(now.Weekday()) + 1) % 7
	thisWeekStart := todayStart.AddDate(0, 0, -daysSinceSat)
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)

	todayFrom := todayStart.UTC().Format(time.RFC3339)
	yesterdayFrom := yesterdayStart.UTC().Format(time.RFC3339)
	todayTo := todayStart.AddDate(0, 0, 1).UTC().Format(time.RFC3339)
	thisWeekFrom := thisWeekStart.UTC().Format(time.RFC3339)
	lastWeekFrom := lastWeekStart.UTC().Format(time.RFC3339)
	lastWeekTo := thisWeekStart.UTC().Format(time.RFC3339)

	const (
		pendingToFixStatuses = `('reported', 'pending')`
		fixedByDevStatuses   = `('fixed', 'changed_by_dev', 'waiting_manager')`
		finishedStatuses     = `('done', 'accepted', 'accept', 'rejected', 'reject')`
	)

	countWhere := func(where string, args ...any) (int, error) {
		var n int
		err := h.DB.QueryRow(`SELECT COUNT(1) FROM bug_reports WHERE `+where, args...).Scan(&n)
		return n, err
	}

	totalBugs, err := countWhere(`1 = 1`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	totalPendingToFix, err := countWhere(`status IN ` + pendingToFixStatuses)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	totalFixedByDev, err := countWhere(`status IN ` + fixedByDevStatuses)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	totalFinished, err := countWhere(`status IN ` + finishedStatuses)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	periodCounts := func(from, to string) (bugs int, fixedByDev int, finished int, err error) {
		bugs, err = countWhere(`created_at >= ? AND created_at < ?`, from, to)
		if err != nil {
			return 0, 0, 0, err
		}
		fixedByDev, err = countWhere(`created_at >= ? AND created_at < ? AND status IN `+fixedByDevStatuses, from, to)
		if err != nil {
			return 0, 0, 0, err
		}
		finished, err = countWhere(`created_at >= ? AND created_at < ? AND status IN `+finishedStatuses, from, to)
		return bugs, fixedByDev, finished, err
	}

	todayBugs, todayFixed, todayFinished, err := periodCounts(todayFrom, todayTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	yesterdayBugs, yesterdayFixed, yesterdayFinished, err := periodCounts(yesterdayFrom, todayFrom)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	thisWeekBugs, thisWeekFixed, thisWeekFinished, err := periodCounts(thisWeekFrom, todayTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
		lastWeekBugs, lastWeekFixed, lastWeekFinished, err := periodCounts(lastWeekFrom, lastWeekTo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}

		byPage, err := h.bugsByPageGroup("")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total": gin.H{
				"bugs":            totalBugs,
				"pending_to_fix": totalPendingToFix,
				"fixed_by_dev":    totalFixedByDev,
				"finished":        totalFinished,
			},
			"today": gin.H{
				"bugs":         todayBugs,
				"fixed_by_dev": todayFixed,
				"finished":     todayFinished,
			},
			"yesterday": gin.H{
				"bugs":         yesterdayBugs,
				"fixed_by_dev": yesterdayFixed,
				"finished":     yesterdayFinished,
			},
			"this_week": gin.H{
				"bugs":         thisWeekBugs,
				"fixed_by_dev": thisWeekFixed,
				"finished":     thisWeekFinished,
			},
			"last_week": gin.H{
				"bugs":         lastWeekBugs,
				"fixed_by_dev": lastWeekFixed,
				"finished":     lastWeekFinished,
			},
			"by_page": byPage,
		})
}

func pageGroupFromURL(raw string) string {
	lower := strings.ToLower(raw)
	marker := "/pages/"
	i := strings.Index(lower, marker)
	start := 0
	if i >= 0 {
		start = i + len(marker)
	} else {
		i = strings.Index(lower, "pages/")
		if i < 0 {
			return "Other"
		}
		start = i + len("pages/")
	}
	if start > len(raw) {
		return "Other"
	}
	rest := raw[start:]
	end := len(rest)
	for j := 0; j < len(rest); j++ {
		ch := rest[j]
		if ch == '/' || ch == '?' || ch == '#' {
			end = j
			break
		}
	}
	seg := strings.TrimSpace(rest[:end])
	if seg == "" {
		return "Other"
	}
	return "Pages/" + seg
}

func (h *Handler) bugsByPageGroup(usernameFilter string) ([]gin.H, error) {
	q := `SELECT b.url FROM bug_reports b`
	args := []any{}
	if usernameFilter != "" {
		q += ` JOIN users r ON r.id = b.reporter_user_id WHERE lower(r.username) = lower(?)`
		args = append(args, usernameFilter)
	}
	rows, err := h.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	order := []string{}
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		key := pageGroupFromURL(url)
		if _, ok := counts[key]; !ok {
			order = append(order, key)
		}
		counts[key]++
	}
	type pair struct {
		k string
		n int
	}
	pairs := make([]pair, 0, len(counts))
	for _, k := range order {
		pairs = append(pairs, pair{k, counts[k]})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].k == "Other" && pairs[j].k != "Other" {
			return false
		}
		if pairs[j].k == "Other" && pairs[i].k != "Other" {
			return true
		}
		return pairs[i].n > pairs[j].n
	})
	out := make([]gin.H, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, gin.H{"page": p.k, "bugs": p.n})
	}
	return out, nil
}

func (h *Handler) MyBugStats(c *gin.Context) {
	username := c.GetString(middleware.ContextUsername)
	h.writeUserBugStats(c, username)
}

func (h *Handler) UserBugStats(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid username"})
		return
	}
	var n int
	if err := h.DB.QueryRow(`SELECT COUNT(1) FROM users WHERE lower(username) = lower(?)`, username).Scan(&n); err != nil || n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	h.writeUserBugStats(c, username)
}

func (h *Handler) writeUserBugStats(c *gin.Context, username string) {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		loc = time.FixedZone("IRST", 3*3600+30*60)
	}
	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	daysSinceSat := (int(now.Weekday()) + 1) % 7
	thisWeekStart := todayStart.AddDate(0, 0, -daysSinceSat)
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)

	todayFrom := todayStart.UTC().Format(time.RFC3339)
	yesterdayFrom := yesterdayStart.UTC().Format(time.RFC3339)
	todayTo := todayStart.AddDate(0, 0, 1).UTC().Format(time.RFC3339)
	thisWeekFrom := thisWeekStart.UTC().Format(time.RFC3339)
	lastWeekFrom := lastWeekStart.UTC().Format(time.RFC3339)
	lastWeekTo := thisWeekStart.UTC().Format(time.RFC3339)

	const (
		pendingToFixStatuses = `('reported', 'pending')`
		fixedByDevStatuses   = `('fixed', 'changed_by_dev', 'waiting_manager')`
		finishedStatuses     = `('done', 'accepted', 'accept', 'rejected', 'reject')`
	)

	userClause := `reporter_user_id = (SELECT id FROM users WHERE lower(username) = lower(?))`
	countWhere := func(where string, args ...any) (int, error) {
		var n int
		err := h.DB.QueryRow(`SELECT COUNT(1) FROM bug_reports WHERE `+where, args...).Scan(&n)
		return n, err
	}

	totalBugs, err := countWhere(userClause, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	totalPendingToFix, err := countWhere(userClause+` AND status IN `+pendingToFixStatuses, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	totalFixedByDev, err := countWhere(userClause+` AND status IN `+fixedByDevStatuses, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	totalFinished, err := countWhere(userClause+` AND status IN `+finishedStatuses, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	periodCounts := func(from, to string) (bugs int, fixedByDev int, finished int, err error) {
		bugs, err = countWhere(userClause+` AND created_at >= ? AND created_at < ?`, username, from, to)
		if err != nil {
			return 0, 0, 0, err
		}
		fixedByDev, err = countWhere(userClause+` AND created_at >= ? AND created_at < ? AND status IN `+fixedByDevStatuses, username, from, to)
		if err != nil {
			return 0, 0, 0, err
		}
		finished, err = countWhere(userClause+` AND created_at >= ? AND created_at < ? AND status IN `+finishedStatuses, username, from, to)
		return bugs, fixedByDev, finished, err
	}

	todayBugs, todayFixed, todayFinished, err := periodCounts(todayFrom, todayTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	yesterdayBugs, yesterdayFixed, yesterdayFinished, err := periodCounts(yesterdayFrom, todayFrom)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	thisWeekBugs, thisWeekFixed, thisWeekFinished, err := periodCounts(thisWeekFrom, todayTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	lastWeekBugs, lastWeekFixed, lastWeekFinished, err := periodCounts(lastWeekFrom, lastWeekTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username": username,
		"total": gin.H{
			"bugs":            totalBugs,
			"pending_to_fix": totalPendingToFix,
			"fixed_by_dev":    totalFixedByDev,
			"finished":        totalFinished,
		},
		"today": gin.H{
			"bugs":         todayBugs,
			"fixed_by_dev": todayFixed,
			"finished":     todayFinished,
		},
		"yesterday": gin.H{
			"bugs":         yesterdayBugs,
			"fixed_by_dev": yesterdayFixed,
			"finished":     yesterdayFinished,
		},
		"this_week": gin.H{
			"bugs":         thisWeekBugs,
			"fixed_by_dev": thisWeekFixed,
			"finished":     thisWeekFinished,
		},
		"last_week": gin.H{
			"bugs":         lastWeekBugs,
			"fixed_by_dev": lastWeekFixed,
			"finished":     lastWeekFinished,
		},
	})
}

func (h *Handler) MarkNotificationRead(c *gin.Context) {
	userID := c.GetInt64(middleware.ContextUserID)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	res, err := h.DB.Exec(`UPDATE notifications SET is_read = 1 WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	userID := c.GetInt64(middleware.ContextUserID)
	if _, err := h.DB.Exec(`UPDATE notifications SET is_read = 1 WHERE user_id = ?`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) DownloadAttachment(c *gin.Context) {
	bugID, err1 := strconv.ParseInt(c.Param("id"), 10, 64)
	attID, err2 := strconv.ParseInt(c.Param("attachmentId"), 10, 64)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var original, stored, contentType string
	err := h.DB.QueryRow(`
SELECT original_filename, stored_filename, content_type
FROM bug_attachments WHERE id = ? AND bug_report_id = ?`, attID, bugID).
		Scan(&original, &stored, &contentType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
		return
	}
	path := filepath.Join(h.UploadDir, stored)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, original))
	c.File(path)
}

func (h *Handler) VAPIDPublicKey(c *gin.Context) {
	if h.Push == nil || h.Push.PublicKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "web push not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"publicKey": h.Push.PublicKey})
}

type pushSubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *Handler) PushSubscribe(c *gin.Context) {
	if h.Push == nil || !h.Push.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "web push not configured"})
		return
	}
	var req pushSubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint and keys are required"})
		return
	}
	userID := c.GetInt64(middleware.ContextUserID)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := h.DB.Exec(`
INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, created_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(endpoint) DO UPDATE SET
  user_id = excluded.user_id,
  p256dh = excluded.p256dh,
  auth = excluded.auth,
  created_at = excluded.created_at`,
		userID, req.Endpoint, req.Keys.P256dh, req.Keys.Auth, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "subscribe failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) PushUnsubscribe(c *gin.Context) {
	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Endpoint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint is required"})
		return
	}
	userID := c.GetInt64(middleware.ContextUserID)
	_, err := h.DB.Exec(`DELETE FROM push_subscriptions WHERE user_id = ? AND endpoint = ?`, userID, req.Endpoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unsubscribe failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func userIDsByRole(q querier, role string) ([]int64, error) {
	rows, err := q.Query(`SELECT id FROM users WHERE role = ?`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func nullStr(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func nullInt(ni sql.NullInt64) interface{} {
	if ni.Valid {
		return ni.Int64
	}
	return nil
}

func randomName() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
