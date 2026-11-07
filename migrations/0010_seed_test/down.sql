DELETE FROM chapters WHERE series_id IN (SELECT id FROM series WHERE slug IN ('naruto','one-piece'));
DELETE FROM series WHERE slug IN ('naruto','one-piece');
