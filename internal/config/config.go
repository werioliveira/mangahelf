package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Env             string
	Port            string
	DatabaseURL     string
	DataDir         string
	StorageDriver   string
	SearchDriver    string
	ImagePipeline   string
	CacheDriver     string
	AuthDriver      string
	SessionSecret   string
	VIPURLTTLMin    int
	S3Endpoint      string
	S3Region        string
	S3Bucket        string
	S3AccessKey     string
	S3SecretKey     string
	S3ForcePathStyle bool
	S3PublicBaseURL string
	ImgproxyBaseURL string
	ImgproxyKey     string
	ImgproxySalt    string
	MeilisearchURL  string
	MeilisearchAPIKey string
	// OIDC / OAuth2 providers
	OIDCGoogleClientID     string
	OIDCGoogleClientSecret string
	OIDCGitHubClientID     string
	OIDCGitHubClientSecret string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
}

var required = []string{
	"SEARCH_DRIVER", "IMAGE_PIPELINE", "STORAGE_DRIVER",
	"CACHE_DRIVER", "AUTH_DRIVER",
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:            getenv("APP_ENV", "development"),
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    getenv("DATABASE_URL", ""),
		DataDir:        getenv("DATA_DIR", "./data"),
		StorageDriver:  getenv("STORAGE_DRIVER", "s3"),
		SearchDriver:   getenv("SEARCH_DRIVER", "postgres"),
		ImagePipeline:  getenv("IMAGE_PIPELINE", "imgproxy"),
		CacheDriver:    getenv("CACHE_DRIVER", "memory"),
		AuthDriver:     getenv("AUTH_DRIVER", "builtin"),
		SessionSecret:  getenv("SESSION_SECRET", ""),
		VIPURLTTLMin:   mustAtoi(getenv("VIP_URL_TTL_MIN", "10")),
		S3Endpoint:     getenv("S3_ENDPOINT", ""),
		S3Region:       getenv("S3_REGION", "auto"),
		S3Bucket:       getenv("S3_BUCKET", ""),
		S3AccessKey:    getenv("S3_ACCESS_KEY", ""),
		S3SecretKey:    getenv("S3_SECRET_KEY", ""),
		S3ForcePathStyle: getenv("S3_FORCE_PATH_STYLE", "true") == "true",
		S3PublicBaseURL:  getenv("S3_PUBLIC_BASE_URL", ""),
		ImgproxyBaseURL:  getenv("IMGPROXY_BASE_URL", ""),
		ImgproxyKey:      getenv("IMGPROXY_KEY", ""),
		ImgproxySalt:     getenv("IMGPROXY_SALT", ""),
		MeilisearchURL:   getenv("MEILISEARCH_URL", ""),
		MeilisearchAPIKey:getenv("MEILISEARCH_API_KEY", ""),
		// OIDC / OAuth2 providers
		OIDCGoogleClientID:     getenv("OIDC_GOOGLE_CLIENT_ID", ""),
		OIDCGoogleClientSecret: getenv("OIDC_GOOGLE_CLIENT_SECRET", ""),
		OIDCGitHubClientID:     getenv("OIDC_GITHUB_CLIENT_ID", ""),
		OIDCGitHubClientSecret: getenv("OIDC_GITHUB_CLIENT_SECRET", ""),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	for _, k := range required {
		if os.Getenv(k) == "" {
			return fmt.Errorf("config: variavel obrigatoria nao definida: %s", k)
		}
	}
	validDrivers := map[string][]string{
		"SEARCH_DRIVER":    {"postgres", "meilisearch"},
		"IMAGE_PIPELINE":   {"imgproxy", "static"},
		"STORAGE_DRIVER":   {"local", "s3"},
		"CACHE_DRIVER":     {"memory", "redis"},
		"AUTH_DRIVER":      {"builtin", "oidc"},
	}
	for k, allowed := range validDrivers {
		v := os.Getenv(k)
		if !contains(allowed, v) {
			return fmt.Errorf("config: %s=%q invalido (opcoes: %v)", k, v, allowed)
		}
	}
	if c.StorageDriver == "s3" && c.S3Bucket == "" {
		return fmt.Errorf("config: S3_BUCKET e obrigatorio quando STORAGE_DRIVER=s3")
	}
	if c.AuthDriver == "builtin" && c.SessionSecret == "" {
		return fmt.Errorf("config: SESSION_SECRET e obrigatorio quando AUTH_DRIVER=builtin")
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustAtoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
