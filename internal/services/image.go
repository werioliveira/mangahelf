package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/mangashelf/mangashelf/internal/config"
	"github.com/mangashelf/mangashelf/internal/ports"
)

type Page struct {
	ID          string    `json:"id"`
	ChapterID   string    `json:"chapter_id"`
	Number      int       `json:"number"`
	StorageKey  string    `json:"storage_key"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	CreatedAt   string    `json:"created_at"`
}

type PageService struct {
	db *sql.DB
}

func NewPageService(db *sql.DB) *PageService {
	return &PageService{db: db}
}

func (s *PageService) List(ctx context.Context, chapterID string) ([]Page, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, chapter_id, number, storage_key, width, height, created_at FROM pages WHERE chapter_id=$1 ORDER BY number ASC",
		chapterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.ChapterID, &p.Number, &p.StorageKey, &p.Width, &p.Height, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type ImageURLBuilder struct {
	cfg *config.Config
}

func NewImageURLBuilder(cfg *config.Config) *ImageURLBuilder {
	return &ImageURLBuilder{cfg: cfg}
}

// variant → largura permitida (imgproxy) ou sufixo (static)
var variantWidth = map[string]int{
	"thumb":    240,
	"medium":   480,
	"full":     800,
	"large":    1200,
	"original": 0,
}

func (b *ImageURLBuilder) Build(ctx context.Context, page ports.Page, variant string) (string, error) {
	w, ok := variantWidth[variant]
	if !ok {
		return "", fmt.Errorf("variante nao suportada: %s", variant)
	}

	switch b.cfg.ImagePipeline {
	case "static":
		return b.buildStatic(page, variant, w)
	case "imgproxy":
		return b.buildImgproxy(page, variant, w)
	default:
		return "", fmt.Errorf("pipeline de imagem invalido: %s", b.cfg.ImagePipeline)
	}
}

func (b *ImageURLBuilder) buildStatic(page ports.Page, variant string, width int) (string, error) {
	base := b.cfg.S3PublicBaseURL
	if base == "" {
		// storage local: URL relativa servida pela API
		return fmt.Sprintf("/api/pages/%s?variant=%s", page.StorageKey, variant), nil
	}
	// S3/CDN: chave pré-renderizada no upload (modo static gera variantes)
	key := page.StorageKey
	if width > 0 {
		key = variantKey(page.StorageKey, variant, width)
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = u.Path + "/" + key
	return u.String(), nil
}

func (b *ImageURLBuilder) buildImgproxy(page ports.Page, variant string, width int) (string, error) {
	if b.cfg.ImgproxyBaseURL == "" {
		return "", fmt.Errorf("IMGPROXY_BASE_URL nao definida")
	}

	// imgproxy path: /<width>x<height>/<storage_key>?... com assinatura HMAC
	// path = /<size>/<key> ; signature = HMAC-SHA256(key/salt, path+expires)
	size := fmt.Sprintf("%dx0", width) // 0 = altura automática (proporção mantida)
	path := fmt.Sprintf("/%s/%s", size, page.StorageKey)

	expires := strconv.FormatInt(time.Now().Add(15*time.Minute).Unix(), 10)
	msg := path + expires

	mac := hmac.New(sha256.New, []byte(b.cfg.ImgproxyKey+b.cfg.ImgproxySalt))
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))

	u, _ := url.Parse(b.cfg.ImgproxyBaseURL)
	u.Path = u.Path + path
	q := u.Query()
	q.Set("expires", expires)
	q.Set("sig", sig)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// variantKey monta a chave da variante pré-renderizada (modo static).
func variantKey(storageKey string, variant string, width int) string {
	// storageKey ex.: pages/{series_id}/{chapter_id}/{hash}.jpg
	// variante ex.: pages/{series_id}/{chapter_id}/{hash}.thumb.240w.jpg
	ext := ""
	if i := lastDotIndex(storageKey); i >= 0 {
		ext = storageKey[i:]
		storageKey = storageKey[:i]
	}
	return fmt.Sprintf("%s.%s.%dw%s", storageKey, variant, width, ext)
}

func lastDotIndex(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

var _ ports.ImageURLBuilder = (*ImageURLBuilder)(nil)

var _ = errors.New
