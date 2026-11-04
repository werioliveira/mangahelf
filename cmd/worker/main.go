package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/mangashelf/mangashelf/internal/adapters/storage/local"
	"github.com/mangashelf/mangashelf/internal/adapters/storage/s3"
	"github.com/mangashelf/mangashelf/internal/config"
	"github.com/mangashelf/mangashelf/internal/ports"
)

const (
	pollInterval = 5 * time.Second
	pollLimit    = 5
)

type chapterJob struct {
	ID          string
	SeriesID    string
	Number      int
	StorageKey  string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config invalido", "err", err)
		os.Exit(1)
	}

	var db *sql.DB
	if cfg.DatabaseURL != "" {
		db, err = sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			slog.Error("falha ao abrir banco", "err", err)
			os.Exit(1)
		}
		defer db.Close()
		if err := db.Ping(); err != nil {
			slog.Error("falha ao pingar banco", "err", err)
			os.Exit(1)
		}
		slog.Info("banco conectado")
	} else {
		slog.Warn("DATABASE_URL nao definida; worker nao processara jobs")
	}

	var storage ports.Storage
	switch cfg.StorageDriver {
	case "s3":
		st, err := s3.New(s3.S3Config{
			Endpoint:        cfg.S3Endpoint,
			Region:          cfg.S3Region,
			Bucket:          cfg.S3Bucket,
			AccessKey:       cfg.S3AccessKey,
			SecretKey:       cfg.S3SecretKey,
			ForcePathStyle:  cfg.S3ForcePathStyle,
		})
		if err != nil {
			slog.Error("falha ao inicializar storage s3", "err", err)
			os.Exit(1)
		}
		storage = st
		slog.Info("storage s3 inicializado", "bucket", cfg.S3Bucket)
	case "local":
		storage = local.New(cfg.DataDir)
		slog.Info("storage local inicializado", "dir", cfg.DataDir)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("worker iniciado", "driver", cfg.StorageDriver, "pipeline", cfg.ImagePipeline)

	ctx := context.Background()
	for {
		if db == nil {
			time.Sleep(pollInterval)
			continue
		}

		jobs, err := fetchJobs(ctx, db)
		if err != nil {
			logger.Error("falha ao buscar jobs", "err", err)
			time.Sleep(pollInterval)
			continue
		}

		for _, job := range jobs {
			if err := processJob(ctx, logger, db, storage, cfg, job); err != nil {
				logger.Error("falha ao processar job", "chapter", job.ID, "err", err)
			}
		}

		time.Sleep(pollInterval)
	}
}

func fetchJobs(ctx context.Context, db *sql.DB) ([]chapterJob, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, series_id, number, storage_key FROM chapters
		 WHERE state='processing' AND storage_key IS NOT NULL AND storage_key != ''
		 ORDER BY created_at ASC
		 LIMIT $1 FOR UPDATE SKIP LOCKED`,
		pollLimit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []chapterJob
	for rows.Next() {
		var j chapterJob
		if err := rows.Scan(&j.ID, &j.SeriesID, &j.Number, &j.StorageKey); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func processJob(ctx context.Context, logger *slog.Logger, db *sql.DB, storage ports.Storage, cfg *config.Config, job chapterJob) error {
	logger.Info("processando capítulo", "id", job.ID, "series", job.SeriesID, "number", job.Number)

	// 1. Baixa o ZIP do storage
	rc, err := storage.Get(ctx, job.StorageKey)
	if err != nil {
		return fmt.Errorf("baixar zip: %w", err)
	}
	defer rc.Close()

	zipData, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("ler zip: %w", err)
	}

	// 2. Extrai imagens do ZIP, ordenadas por nome
	files, err := extractZipImages(zipData)
	if err != nil {
		return fmt.Errorf("extrair zip: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("zip sem imagens")
	}

	// 3. Grava cada página no storage com chave baseada em hash de conteúdo
	for i, f := range files {
		hash := sha256.Sum256(f.data)
		hashStr := hex.EncodeToString(hash[:])
		ext := strings.ToLower(filepath.Ext(f.name))
		if ext == "" {
			ext = ".bin"
		}
		key := fmt.Sprintf("pages/%s/%d/%d_%s%s", job.SeriesID, job.Number, i+1, hashStr, ext)

		if err := storage.Put(ctx, key, bytes.NewReader(f.data), int64(len(f.data))); err != nil {
			return fmt.Errorf("gravar pagina %d: %w", i+1, err)
		}
	}

	// 4. Publica o capítulo
	if _, err := db.ExecContext(ctx, `UPDATE chapters SET state='published' WHERE id=$1`, job.ID); err != nil {
		return fmt.Errorf("publicar capitulo: %w", err)
	}

	logger.Info("capitulo publicado", "id", job.ID, "paginas", len(files))
	return nil
}

type zipFile struct {
	name string
	data []byte
}

func extractZipImages(data []byte) ([]zipFile, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var files []zipFile
	for _, f := range reader.File {
		if isImageFile(f.Name) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}
			files = append(files, zipFile{name: f.Name, data: content})
		}
	}

	// ordena por nome para pagina 1, 2, 3...
	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})
	return files, nil
}

func isImageFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".webp")
}

var _ = ports.OpPut
