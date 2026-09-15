package config

import (
	"os"
)

type Config struct {
	Addr           string
	DatabasePath   string
	UploadDir      string
	JWTSecret      string
	GinMode        string
	VAPIDPublicKey string
	VAPIDPrivateKey string
	VAPIDSubject   string
	PublicBaseURL  string
}

func Load() Config {
	return Config{
		Addr:            getenv("ADDR", ":8080"),
		DatabasePath:    getenv("DATABASE_PATH", "./data/bug-report.db"),
		UploadDir:       getenv("UPLOAD_DIR", "./data/uploads"),
		JWTSecret:       getenv("JWT_SECRET", "raven-dev-secret"),
		GinMode:         getenv("GIN_MODE", "debug"),
		VAPIDPublicKey:  getenv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey: getenv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:    getenv("VAPID_SUBJECT", "mailto:bugs@raven.local"),
		PublicBaseURL:   getenv("PUBLIC_BASE_URL", "https://pc-armin:8443/bugs"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
