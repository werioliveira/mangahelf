-- Seed test data: sample series and chapters
INSERT INTO series (title, slug, synopsis, type, status, access, created_by)
SELECT 'Naruto', 'naruto', 'Ninja em treinamento.', 'manga', 'ongoing', 'public',
       (SELECT id FROM users WHERE email='admin@mangashelf.local')
WHERE NOT EXISTS (SELECT 1 FROM series WHERE slug='naruto');

INSERT INTO series (title, slug, synopsis, type, status, access, created_by)
SELECT 'One Piece', 'one-piece', 'Piratas em busca do tesouro.', 'manga', 'ongoing', 'public',
       (SELECT id FROM users WHERE email='admin@mangashelf.local')
WHERE NOT EXISTS (SELECT 1 FROM series WHERE slug='one-piece');

-- chapters for naruto
INSERT INTO chapters (series_id, number, title, state, access)
SELECT s.id, 1, 'Capitulo 1', 'published', 'public'
FROM series s WHERE s.slug='naruto'
AND NOT EXISTS (SELECT 1 FROM chapters c WHERE c.series_id=s.id AND c.number=1);

INSERT INTO chapters (series_id, number, title, state, access)
SELECT s.id, 2, 'Capitulo 2', 'draft', 'public'
FROM series s WHERE s.slug='naruto'
AND NOT EXISTS (SELECT 1 FROM chapters c WHERE c.series_id=s.id AND c.number=2);

-- chapters for one-piece
INSERT INTO chapters (series_id, number, title, state, access)
SELECT s.id, 1, 'Capitulo 1', 'published', 'public'
FROM series s WHERE s.slug='one-piece'
AND NOT EXISTS (SELECT 1 FROM chapters c WHERE c.series_id=s.id AND c.number=1);
