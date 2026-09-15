package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/ArminDashti/raven-api/internal/config"
	appdb "github.com/ArminDashti/raven-api/internal/db"
	"github.com/ArminDashti/raven-api/internal/handlers"
	"github.com/ArminDashti/raven-api/internal/middleware"
	"github.com/ArminDashti/raven-api/internal/push"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("upload dir: %v", err)
	}

	sqlDB, err := appdb.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer sqlDB.Close()

	if err := appdb.Migrate(sqlDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := appdb.SeedUsers(sqlDB); err != nil {
		log.Fatalf("seed: %v", err)
	}

	pushSender := &push.Sender{
		DB:         sqlDB,
		PublicKey:  cfg.VAPIDPublicKey,
		PrivateKey: cfg.VAPIDPrivateKey,
		Subject:    cfg.VAPIDSubject,
		BaseURL:    strings.TrimRight(cfg.PublicBaseURL, "/"),
	}
	if !pushSender.Enabled() {
		log.Printf("web push disabled: set VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY")
	}

	h := handlers.New(sqlDB, cfg.JWTSecret, cfg.UploadDir, pushSender)
	r := gin.Default()
	r.MaxMultipartMemory = 64 << 20 // 64 MiB
	r.Use(cors.New(cors.Config{
		AllowOriginFunc:  func(string) bool { return true },
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", h.Health)

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", h.Login)

		authz := api.Group("")
		authz.Use(middleware.RequireAuth(cfg.JWTSecret))
		{
			authz.GET("/auth/me", h.Me)
			authz.PATCH("/auth/me", h.UpdateMe)
			authz.POST("/auth/me/avatar", h.UploadAvatar)
			authz.GET("/users", h.ListUsers)
			authz.GET("/users/:id/avatar", h.DownloadUserAvatar)
			// Gin: same wildcard name as /users/:id/avatar (cannot mix :id and :username).
			authz.GET("/users/:id", h.GetUserByUsername)
			authz.GET("/bug-reports", h.ListBugReports)
			authz.GET("/stats", h.BugStats)
			authz.GET("/stats/me", h.MyBugStats)
			authz.GET("/stats/user/:username", h.UserBugStats)
			authz.POST("/bug-reports", h.CreateBugReport)
			authz.GET("/bug-reports/:id", h.GetBugReport)
			authz.PATCH("/bug-reports/:id", h.UpdateBugFields)
			authz.DELETE("/bug-reports/:id", h.DeleteBugReport)
			authz.PATCH("/bug-reports/:id/status", h.UpdateBugStatus)
			authz.GET("/bug-reports/:id/history", h.BugHistory)
			authz.GET("/bug-reports/:id/team-chat", h.ListTeamMessages)
			authz.POST("/bug-reports/:id/team-chat", h.CreateTeamMessage)
			authz.GET("/bug-reports/:id/attachments", h.ListAttachments)
			authz.GET("/bug-reports/:id/attachments/:attachmentId", h.DownloadAttachment)
			authz.GET("/notifications", h.ListNotifications)
			authz.GET("/notifications/unread-count", h.UnreadCount)
			authz.PATCH("/notifications/:id/read", h.MarkNotificationRead)
			authz.POST("/notifications/read-all", h.MarkAllNotificationsRead)
			authz.GET("/push/vapid-public-key", h.VAPIDPublicKey)
			authz.POST("/push/subscribe", h.PushSubscribe)
			authz.DELETE("/push/subscribe", h.PushUnsubscribe)
		}
	}

	log.Printf("raven-api listening on %s", cfg.Addr)
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
