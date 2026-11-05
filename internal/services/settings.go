package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type Setting struct {
	Key         string      `json:"key"`
	Value       json.RawMessage `json:"value"`
	Type        string      `json:"type"`
	SettingGroup string     `json:"setting_group"`
	UpdatedAt   string      `json:"updated_at"`
}

type SettingService struct {
	db     *sql.DB
	mu     sync.RWMutex
	cache  map[string]Setting
	ttl    time.Duration
}

func NewSettingService(db *sql.DB) *SettingService {
	return &SettingService{
		db:   db,
		cache: make(map[string]Setting),
		ttl:   60 * time.Second,
	}
}

// Get retorna uma configuração pela chave. Busca no cache primeiro; se não
// encontrar ou expirar, consulta o banco de dados.
func (s *SettingService) Get(ctx context.Context, key string) (Setting, bool, error) {
	s.mu.RLock()
	setting, ok := s.cache[key]
	s.mu.RUnlock()
	if ok {
		return setting, true, nil
	}

	var value json.RawMessage
	var typ, group, updatedAt string
	err := s.db.QueryRowContext(ctx,
		"SELECT value, type, setting_group, updated_at FROM settings WHERE key=$1",
		key,
	).Scan(&value, &typ, &group, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Setting{}, false, nil
		}
		return Setting{}, false, err
	}

	setting = Setting{
		Key:          key,
		Value:        value,
		Type:         typ,
		SettingGroup: group,
		UpdatedAt:    updatedAt,
	}
	s.mu.Lock()
	s.cache[key] = setting
	s.mu.Unlock()

	return setting, true, nil
}

// GetString retorna o valor de uma configuração como string (desencapsulando aspas JSON).
func (s *SettingService) GetString(ctx context.Context, key string) (string, bool, error) {
	setting, ok, err := s.Get(ctx, key)
	if err != nil || !ok {
		return "", false, err
	}
	var str string
	if err := json.Unmarshal(setting.Value, &str); err != nil {
		return "", false, err
	}
	return str, true, nil
}

// GetBool retorna o valor de uma configuração como booleano.
func (s *SettingService) GetBool(ctx context.Context, key string) (bool, bool, error) {
	setting, ok, err := s.Get(ctx, key)
	if err != nil || !ok {
		return false, false, err
	}
	var b bool
	if err := json.Unmarshal(setting.Value, &b); err != nil {
		return false, false, err
	}
	return b, true, nil
}

// GetGroup retorna todas as configurações de um grupo.
func (s *SettingService) GetGroup(ctx context.Context, group string) ([]Setting, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT key, value, type, setting_group, updated_at FROM settings WHERE setting_group=$1 ORDER BY key",
		group,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Setting
	for rows.Next() {
		var setting Setting
		if err := rows.Scan(&setting.Key, &setting.Value, &setting.Type, &setting.SettingGroup, &setting.UpdatedAt); err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.cache[setting.Key] = setting
		s.mu.Unlock()
		out = append(out, setting)
	}
	return out, rows.Err()
}

// Set cria ou atualiza uma configuração e invalida o cache.
func (s *SettingService) Set(ctx context.Context, key, typ, group string, value interface{}) error {
	valBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO settings (key, value, type, setting_group, updated_at)
		 VALUES ($1, $2, $3, $4, now())
		 ON CONFLICT (key) DO UPDATE
		 SET value = EXCLUDED.value,
		     type = EXCLUDED.type,
		     setting_group = EXCLUDED.setting_group,
		     updated_at = now()`,
		key, valBytes, typ, group,
	)
	if err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()

	return nil
}

// Invalidate remove uma chave específica do cache.
func (s *SettingService) Invalidate(key string) {
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()
}

// IsStaff verifica se o usuário tem permissão de moderação (staff).
func (s *SettingService) IsStaff(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(
		 SELECT 1 FROM user_roles ur
		 JOIN role_permissions rp ON rp.role_id = ur.role_id
		 WHERE ur.user_id = $1
		   AND rp.permission_key IN ('chapter.publish','chapter.set_access','media.manage','report.manage')
		)`,
		userID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// Reload recarrega todas as configurações do banco para o cache.
func (s *SettingService) Reload(ctx context.Context) error {
	s.mu.Lock()
	s.cache = make(map[string]Setting)
	s.mu.Unlock()

	rows, err := s.db.QueryContext(ctx,
		"SELECT key, value, type, setting_group, updated_at FROM settings",
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var setting Setting
		if err := rows.Scan(&setting.Key, &setting.Value, &setting.Type, &setting.SettingGroup, &setting.UpdatedAt); err != nil {
			return err
		}
		s.mu.Lock()
		s.cache[setting.Key] = setting
		s.mu.Unlock()
	}
	return nil
}

var _ = errors.New
