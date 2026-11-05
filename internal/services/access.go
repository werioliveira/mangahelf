package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AccessResolver resolve acesso a capítulos (resolver único — seção 5.2 do roadmap).
// Nunca checa "é VIP" diretamente; toda decisão passa por CanRead.
type AccessResolver struct {
	db *sql.DB
}

func NewAccessResolver(db *sql.DB) *AccessResolver {
	return &AccessResolver{db: db}
}

// chapterInfo carrega as informações de acesso de um capítulo (incluindo a série).
type chapterInfo struct {
	ID          string
	SeriesID    string
	SeriesAccess string // 'public' | 'vip_only'
	Access      string // 'public' | 'early' | 'vip_only'
	PublicAt    *time.Time
	GroupID     string
}

func (r *AccessResolver) loadChapter(ctx context.Context, chapterID string) (*chapterInfo, error) {
	var c chapterInfo
	var publicAt sql.NullTime
	var groupID sql.NullString
	var seriesAccess sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT c.id, c.series_id, c.access, c.public_at, s.group_id, s.access
		 FROM chapters c
		 JOIN series s ON s.id = c.series_id
		 WHERE c.id=$1`,
		chapterID,
	).Scan(&c.ID, &c.SeriesID, &c.Access, &publicAt, &groupID, &seriesAccess)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("capitulo nao encontrado")
		}
		return nil, err
	}
	if publicAt.Valid {
		c.PublicAt = &publicAt.Time
	}
	if groupID.Valid {
		c.GroupID = groupID.String
	}
	if seriesAccess.Valid {
		c.SeriesAccess = seriesAccess.String
	}
	return &c, nil
}

// HasActiveVIP verifica se o usuário tem uma assinatura VIP ativa (expires_at > now).
func (r *AccessResolver) HasActiveVIP(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM vip_memberships
		 WHERE user_id=$1 AND expires_at > now())`,
		userID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// isStaffOrEditor verifica se o usuário tem permissão de editar ou moderar o grupo
// (qualquer permissão de capítulo ou mídia no escopo do grupo).
func (r *AccessResolver) isStaffOrEditor(ctx context.Context, userID string, groupID string) (bool, error) {
	// Verifica se o usuário tem alguma permissão atômica de edição/moderação no grupo
	// (chapter.publish, chapter.set_access, media.manage, report.handle, etc.)
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(
		 SELECT 1 FROM user_roles ur
		 JOIN role_permissions rp ON rp.role_id = ur.role_id
		 JOIN group_members gm ON gm.user_id = ur.user_id AND gm.group_id = $2
		 WHERE ur.user_id = $1
		   AND rp.permission_key IN ('chapter.publish','chapter.set_access','media.manage','report.handle')
		)`,
		userID, groupID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	// Também checa user_roles globais (escopo vazio = global)
	err = r.db.QueryRowContext(ctx,
		`SELECT EXISTS(
		 SELECT 1 FROM user_roles ur
		 JOIN role_permissions rp ON rp.role_id = ur.role_id
		 WHERE ur.user_id = $1
		   AND rp.permission_key IN ('chapter.publish','chapter.set_access','media.manage','report.handle')
		)`,
		userID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// CanRead retorna true se o usuário pode ler o capítulo.
// É o resolver único usado pela API, pelo leitor, pela busca e pelos feeds.
func (r *AccessResolver) CanRead(ctx context.Context, userID string, chapterID string) (bool, error) {
	ch, err := r.loadChapter(ctx, chapterID)
	if err != nil {
		return false, err
	}

	// Equipe/editor vê sem ser VIP
	if ch.GroupID != "" {
		ok, err := r.isStaffOrEditor(ctx, userID, ch.GroupID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	switch ch.Access {
	case "public":
		return true, nil
	case "early":
		// VIP não espera
		if userID != "" {
			ok, err := r.HasActiveVIP(ctx, userID)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		// Demais só a partir de public_at
		if ch.PublicAt != nil && !time.Now().Before(*ch.PublicAt) {
			return true, nil
		}
		return false, nil
	case "vip_only":
		// Se a série é vip_only, todos os capítulos herdam a regra
		if ch.SeriesAccess == "vip_only" {
			if userID != "" {
				ok, err := r.HasActiveVIP(ctx, userID)
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
			return false, nil
		}
		// Capítulo vip_only com public_at opcional
		if userID != "" {
			ok, err := r.HasActiveVIP(ctx, userID)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		if ch.PublicAt != nil && !time.Now().Before(*ch.PublicAt) {
			return true, nil
		}
		return false, nil
	default:
		return false, nil
	}
}

// CanReadChapter é um atalho que recebe o ID do capítulo e retorna o chapterInfo + decisão.
func (r *AccessResolver) CanReadChapter(ctx context.Context, userID string, chapterID string) (*chapterInfo, bool, error) {
	can, err := r.CanRead(ctx, userID, chapterID)
	if err != nil {
		return nil, false, err
	}
	ch, err := r.loadChapter(ctx, chapterID)
	if err != nil {
		return nil, false, err
	}
	return ch, can, nil
}

var _ = fmt.Sprintf
