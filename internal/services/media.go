package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/mangashelf/mangashelf/internal/config"
	"github.com/mangashelf/mangashelf/internal/ports"
)

type MediaService struct {
	db      *sql.DB
	storage ports.Storage
	cfg     *config.Config
	authSvc *AuthService
}

type PresignUploadInput struct {
	SeriesID    string `json:"series_id"`
	ChapterNumber int  `json:"chapter_number"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

type PresignUploadOutput struct {
	UploadURL   string `json:"upload_url"`
	StorageKey  string `json:"storage_key"`
	Method      string `json:"method"` // "PUT"
	ExpiresIn   int    `json:"expires_in"` // seconds
}

type CreateUploadInput struct {
	SeriesID     string `json:"series_id"`
	ChapterNumber int   `json:"chapter_number"`
	StorageKey   string `json:"storage_key"`
	Title        string `json:"title,omitempty"`
}

type CreateUploadOutput struct {
	ChapterID string `json:"chapter_id"`
	Status    string `json:"status"` // "processing"
}

const uploadTTL = 15 * time.Minute

func NewMediaService(db *sql.DB, storage ports.Storage, cfg *config.Config, authSvc *AuthService) *MediaService {
	return &MediaService{db: db, storage: storage, cfg: cfg, authSvc: authSvc}
}

func (m *MediaService) contentKey(filename string) string {
	buf := make([]byte, 16)
	rand.Read(buf)
	ext := filepath.Ext(filename)
	return fmt.Sprintf("uploads/%s%s", hex.EncodeToString(buf), ext)
}

func (m *MediaService) PresignUpload(ctx context.Context, userID string, in PresignUploadInput) (PresignUploadOutput, error) {
	if userID == "" {
		return PresignUploadOutput{}, errors.New("usuario nao autenticado")
	}
	if in.SeriesID == "" {
		return PresignUploadOutput{}, errors.New("series_id e obrigatorio")
	}
	if in.ChapterNumber <= 0 {
		return PresignUploadOutput{}, errors.New("chapter_number deve ser maior que zero")
	}
	if in.Filename == "" {
		return PresignUploadOutput{}, errors.New("filename e obrigatorio")
	}

	// validate content type
	if in.ContentType != "" {
		exts, _ := mime.ExtensionsByType(in.ContentType)
		if len(exts) == 0 {
			return PresignUploadOutput{}, errors.New("content_type nao suportado")
		}
	}

	key := m.contentKey(in.Filename)
	url, err := m.storage.Presign(ctx, key, ports.OpPut, uploadTTL)
	if err != nil {
		return PresignUploadOutput{}, fmt.Errorf("gerar presign: %w", err)
	}

	return PresignUploadOutput{
		UploadURL:   url,
		StorageKey:  key,
		Method:      "PUT",
		ExpiresIn:   int(uploadTTL.Seconds()),
	}, nil
}

func (m *MediaService) CreateUpload(ctx context.Context, userID string, in CreateUploadInput) (CreateUploadOutput, error) {
	if in.SeriesID == "" {
		return CreateUploadOutput{}, errors.New("series_id e obrigatorio")
	}
	if in.ChapterNumber <= 0 {
		return CreateUploadOutput{}, errors.New("chapter_number deve ser maior que zero")
	}
	if in.StorageKey == "" {
		return CreateUploadOutput{}, errors.New("storage_key e obrigatorio")
	}

	// verify series exists
	var exists bool
	err := m.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM series WHERE id=$1)", in.SeriesID).Scan(&exists)
	if err != nil {
		return CreateUploadOutput{}, err
	}
	if !exists {
		return CreateUploadOutput{}, errors.New("serie nao encontrada")
	}

	// check duplicate chapter number in series
	var dup int
	err = m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chapters WHERE series_id=$1 AND number=$2", in.SeriesID, in.ChapterNumber).Scan(&dup)
	if err != nil {
		return CreateUploadOutput{}, err
	}
	if dup > 0 {
		return CreateUploadOutput{}, errors.New("numero de capitulo ja existe nesta serie")
	}

	var id string
	err = m.db.QueryRowContext(ctx,
		"INSERT INTO chapters (series_id, number, title, state, access, storage_key) VALUES ($1, $2, $3, 'processing', 'public', $4) RETURNING id",
		in.SeriesID, in.ChapterNumber, in.Title, in.StorageKey,
	).Scan(&id)
	if err != nil {
		return CreateUploadOutput{}, err
	}

	// TODO: enqueue job in worker (River queue). For now, chapter stays in 'processing'
	// until worker picks it up via polling or a separate job table.

	return CreateUploadOutput{
		ChapterID: id,
		Status:    "processing",
	}, nil
}

func (m *MediaService) ValidateMagicBytes(data []byte) error {
	if len(data) < 4 {
		return errors.New("arquivo muito pequeno")
	}
	// ZIP magic: PK\x03\x14 or PK\x05\x06 (empty zip) or PK\x07\x08 (spanned)
	if bytesEqualPrefix(data, []byte{0x50, 0x4B, 0x03, 0x04}) ||
		bytesEqualPrefix(data, []byte{0x50, 0x4B, 0x05, 0x06}) ||
		bytesEqualPrefix(data, []byte{0x50, 0x4B, 0x07, 0x08}) {
		return nil
	}
	// image magic bytes
	if bytesEqualPrefix(data, []byte{0xFF, 0xD8, 0xFF}) { // JPEG
		return nil
	}
	if bytesEqualPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) { // PNG
		return nil
	}
	if bytesEqualPrefix(data, []byte{0x52, 0x49, 0x46, 0x46}) && strings.HasSuffix(string(data[8:12]), "WEBP") { // WebP
		return nil
	}
	return errors.New("tipo de arquivo nao suportado (esperado ZIP, JPEG, PNG ou WebP)")
}

func bytesEqualPrefix(a, b []byte) bool {
	if len(a) < len(b) {
		return false
	}
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}