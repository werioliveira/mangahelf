-- Migration: 0001_core
-- Schema inicial coerente com o código atual

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Grupos
CREATE TABLE groups (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    slug        text UNIQUE NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- Usuários
CREATE TABLE users (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email             citext NOT NULL UNIQUE,
    email_verified_at timestamptz,
    username          citext UNIQUE,
    name              text,
    password_hash     text,
    status            text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended','banned','pending_deletion')),
    oauth_provider    text,
    oauth_provider_id text,
    failed_login_attempts int NOT NULL DEFAULT 0,
    locked_until      timestamptz,
    last_login_at     timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_oauth_provider_id ON users(oauth_provider, oauth_provider_id);


-- Identidades externas
CREATE TABLE external_identities (
    provider   text NOT NULL,
    subject    text NOT NULL,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject)
);

-- Sessões
CREATE TABLE sessions (
    token  text PRIMARY KEY,
    data   bytea NOT NULL,
    expiry timestamptz NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions (expiry);

CREATE TABLE user_sessions (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    last_seen_at   timestamptz NOT NULL DEFAULT now(),
    auth_time      timestamptz NOT NULL,
    mfa_at         timestamptz,
    ip             inet,
    user_agent     text,
    revoked_at     timestamptz,
    revoked_reason text
);
CREATE INDEX user_sessions_user_idx ON user_sessions (user_id) WHERE revoked_at IS NULL;


-- Tokens
CREATE TABLE auth_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    text NOT NULL CHECK (purpose IN ('verify_email','reset_password','change_email')),
    token_hash bytea NOT NULL UNIQUE,
    meta       jsonb NOT NULL DEFAULT '{}',
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- MFA
CREATE TABLE user_totp (
    user_id        uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret_enc     bytea NOT NULL,
    key_id         smallint NOT NULL,
    confirmed_at   timestamptz,
    last_used_step bigint NOT NULL DEFAULT 0
);
CREATE TABLE user_recovery_codes (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hmac bytea NOT NULL,
    used_at   timestamptz
);
CREATE TABLE user_webauthn_credentials (
    id            bytea PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key    bytea NOT NULL,
    sign_count    bigint NOT NULL DEFAULT 0,
    transports    text[] NOT NULL DEFAULT '{}',
    aaguid        bytea,
    backup_eligible boolean NOT NULL DEFAULT false,
    backup_state    boolean NOT NULL DEFAULT false,
    name          text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_used_at  timestamptz
);

-- RBAC
CREATE TABLE permissions (
    key           text PRIMARY KEY,
    desc_text     text,
    scopes        text[] NOT NULL DEFAULT '{}',
    requires_mfa  boolean NOT NULL DEFAULT false
);
CREATE TABLE roles (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name      text NOT NULL UNIQUE,
    position  int  NOT NULL DEFAULT 0,
    system    boolean NOT NULL DEFAULT false,
    group_id  uuid REFERENCES groups(id) ON DELETE CASCADE
);
CREATE TABLE role_permissions (
    role_id        uuid REFERENCES roles(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_key)
);
CREATE TABLE user_roles (
    user_id     uuid REFERENCES users(id) ON DELETE CASCADE,
    role_id     uuid REFERENCES roles(id) ON DELETE CASCADE,
    granted_at  timestamptz NOT NULL DEFAULT now(),
    granted_by  uuid REFERENCES users(id),
    PRIMARY KEY (user_id, role_id)
);
CREATE TABLE group_members (
    group_id uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id  uuid REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id, role_id)
);
CREATE TABLE user_overrides (
    user_id        uuid REFERENCES users(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    scope_group_id uuid,
    effect         text NOT NULL CHECK (effect IN ('allow','deny')),
    reason         text,
    created_by     uuid REFERENCES users(id),
    expires_at     timestamptz,
    PRIMARY KEY (user_id, permission_key, scope_group_id)
);


-- Séries/Catálogo
CREATE TABLE series (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title             text NOT NULL,
    slug              text UNIQUE NOT NULL,
    synopsis          text,
    type              text NOT NULL DEFAULT 'manga' CHECK (type IN ('manga', 'manhwa', 'manhua')),
    status            text NOT NULL DEFAULT 'ongoing' CHECK (status IN ('ongoing', 'hiatus', 'completed', 'cancelled')),
    demography        text NOT NULL DEFAULT 'unspecified' CHECK (demography IN ('shounen', 'shoujo', 'seinen', 'josei', 'unspecified')),
    original_language text NOT NULL DEFAULT 'ja' CHECK (original_language IN ('ja', 'ko', 'zh', 'en', 'other')),
    release_year      int CHECK (release_year IS NULL OR release_year BETWEEN 1900 AND 3000),
    access            text NOT NULL DEFAULT 'public' CHECK (access IN ('public','vip_only')),
    publication_status text NOT NULL DEFAULT 'draft' CHECK (publication_status IN ('draft', 'scheduled', 'published', 'removed')),
    published_at      timestamptz,
    public_at         timestamptz,
    cover_storage_key text,
    group_id          uuid REFERENCES groups(id) ON DELETE RESTRICT,
    created_by        uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_series_type ON series(type);
CREATE INDEX idx_series_status ON series(status);
CREATE INDEX idx_series_access ON series(access);
CREATE INDEX idx_series_group ON series(group_id) WHERE group_id IS NOT NULL;

-- Autores/tags/alt titles
CREATE TABLE authors (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    slug       text NOT NULL UNIQUE,
    bio        text NOT NULL DEFAULT '',
    avatar_url text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE series_authors (
    series_id  uuid REFERENCES series(id) ON DELETE CASCADE,
    author_id  uuid REFERENCES authors(id) ON DELETE CASCADE,
    role       text NOT NULL DEFAULT 'author' CHECK (role IN ('author','artist')),
    sort_order int  NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (series_id, author_id, role)
);
CREATE TABLE tags (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    slug       text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_tags_name_lower ON tags (lower(name));
CREATE TABLE series_tags (
    series_id  uuid REFERENCES series(id) ON DELETE CASCADE,
    tag_id     uuid REFERENCES tags(id) ON DELETE CASCADE,
    sort_order int  NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (series_id, tag_id)
);
CREATE TABLE series_alt_titles (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id  uuid NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    title      text NOT NULL CHECK (length(btrim(title)) > 0),
    sort_order int  NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (series_id, title)
);
CREATE INDEX idx_alt_titles_series ON series_alt_titles(series_id);


-- Capítulos
CREATE TABLE chapters (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id        uuid NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    number           int NOT NULL,
    title            text,
    state            text NOT NULL DEFAULT 'draft' CHECK (state IN ('draft','processing','published','removed')),
    access           text NOT NULL DEFAULT 'public' CHECK (access IN ('public','early','vip_only')),
    public_at        timestamptz,
    storage_key      text,
    source_upload_id uuid,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (series_id, number)
);
CREATE INDEX idx_chapters_series_state ON chapters(series_id, state);
CREATE INDEX idx_chapters_public_at ON chapters(public_at);

-- Páginas
CREATE TABLE pages (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id  uuid NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    number      int NOT NULL,
    storage_key text NOT NULL,
    width       int,
    height      int,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (chapter_id, number)
);
CREATE INDEX idx_pages_chapter_number ON pages(chapter_id, number);

CREATE TABLE chapter_pages (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id     uuid NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    page_number    integer NOT NULL CHECK (page_number > 0),
    storage_key    text NOT NULL UNIQUE,
    content_hash   char(64) NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    content_type   text NOT NULL,
    size_bytes     bigint NOT NULL CHECK (size_bytes > 0),
    width          integer NULL CHECK (width IS NULL OR width > 0),
    height         integer NULL CHECK (height IS NULL OR height > 0),
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (chapter_id, page_number)
);
CREATE INDEX idx_chapter_pages_chapter_order ON chapter_pages (chapter_id, page_number);

-- Uploads
CREATE TABLE media_uploads (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    storage_key    text NOT NULL UNIQUE,
    kind           text NOT NULL CHECK (kind IN ('chapter_source','cover_source','avatar_source')),
    status         text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','uploaded','processing','completed','rejected','failed','expired')),
    content_type   text NOT NULL,
    size_bytes     bigint NOT NULL CHECK (size_bytes > 0),
    sha256         char(64) CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    uploaded_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    expires_at     timestamptz,
    processed_at   timestamptz,
    failure_reason text
);
CREATE INDEX idx_media_uploads_kind ON media_uploads(kind);
CREATE INDEX idx_media_uploads_status ON media_uploads(status);

ALTER TABLE chapters ADD CONSTRAINT fk_chapters_source_upload FOREIGN KEY (source_upload_id) REFERENCES media_uploads(id) ON DELETE SET NULL;


 - -   D e f a u l t s 
 I N S E R T   I N T O   s e t t i n g s   ( k e y ,   v a l u e ,   t y p e ,   s e t t i n g _ g r o u p )   V A L U E S 
         ( ' a p p _ n a m e ' ,   ' \  
 M a n g a S h e l f \ ' ,   ' s t r i n g ' ,   ' s i t e ' ) , 
         ( ' s e a r c h _ e n g i n e ' ,   ' \ p o s t g r e s \ ' ,   ' s t r i n g ' ,   ' s e a r c h ' ) , 
         ( ' u i _ p r i m a r y _ c o l o r ' ,   ' \ # 3 b 8 2 f 6 \ ' ,   ' c o l o r ' ,   ' u i ' ) , 
         ( ' u i _ s e c o n d a r y _ c o l o r ' ,   ' \ # 1 e 2 9 3 b \ ' ,   ' c o l o r ' ,   ' u i ' ) , 
         ( ' u i _ a c c e n t _ c o l o r ' ,   ' \ # f 5 9 e 0 b \ ' ,   ' c o l o r ' ,   ' u i ' ) , 
         ( ' s i t e _ d e s c r i p t i o n ' ,   ' \ A  
 m e l h o r  
 p l a t a f o r m a  
 d e  
 m a n g á s  
 e m  
 p o r t u g u ê s \ ' ,   ' s t r i n g ' ,   ' s i t e ' ) 
 O N   C O N F L I C T   ( k e y )   D O   N O T H I N G ; 
 
 D R O P   V I E W   I F   E X I S T S   v _ s e r i e s _ a d m i n ; 
 C R E A T E   V I E W   v _ s e r i e s _ a d m i n   A S 
 S E L E C T   s . i d , s . t i t l e , s . s l u g , s . s y n o p s i s , s . t y p e , s . s t a t u s , s . d e m o g r a p h y , s . o r i g i n a l _ l a n g u a g e , s . r e l e a s e _ y e a r , s . a c c e s s , s . p u b l i c a t i o n _ s t a t u s , s . p u b l i s h e d _ a t , s . p u b l i c _ a t , s . c o v e r _ s t o r a g e _ k e y , s . g r o u p _ i d , s . c r e a t e d _ b y , s . c r e a t e d _ a t , s . u p d a t e d _ a t , 
 C O A L E S C E ( ( S E L E C T   j s o n b _ a g g ( j s o n b _ b u i l d _ o b j e c t ( ' i d ' , a . i d , ' n a m e ' , a . n a m e , ' r o l e ' , s a . r o l e )   O R D E R   B Y   s a . s o r t _ o r d e r )   F R O M   s e r i e s _ a u t h o r s   s a   J O I N   a u t h o r s   a   O N   a . i d = s a . a u t h o r _ i d   W H E R E   s a . s e r i e s _ i d = s . i d ) , ' [ ] ' : : j s o n b )   A S   a u t h o r s , 
 C O A L E S C E ( ( S E L E C T   j s o n b _ a g g ( j s o n b _ b u i l d _ o b j e c t ( ' i d ' , t . i d , ' n a m e ' , t . n a m e )   O R D E R   B Y   s t . s o r t _ o r d e r )   F R O M   s e r i e s _ t a g s   s t   J O I N   t a g s   t   O N   t . i d = s t . t a g _ i d   W H E R E   s t . s e r i e s _ i d = s . i d ) , ' [ ] ' : : j s o n b )   A S   t a g s , 
 C O A L E S C E ( ( S E L E C T   j s o n b _ a g g ( j s o n b _ b u i l d _ o b j e c t ( ' t i t l e ' , s a t . t i t l e )   O R D E R   B Y   s a t . s o r t _ o r d e r )   F R O M   s e r i e s _ a l t _ t i t l e s   s a t   W H E R E   s a t . s e r i e s _ i d = s . i d ) , ' [ ] ' : : j s o n b )   A S   a l t _ t i t l e s , 
 C O U N T ( c . i d ) : : i n t   A S   t o t a l _ c h a p t e r _ c o u n t , 
 C O U N T ( c . i d )   F I L T E R   ( W H E R E   c . s t a t e = ' p u b l i s h e d ' ) : : i n t   A S   p u b l i s h e d _ c h a p t e r _ c o u n t 
 F R O M   s e r i e s   s   L E F T   J O I N   c h a p t e r s   c   O N   c . s e r i e s _ i d = s . i d   G R O U P   B Y   s . i d ; 
  
 