-- A battle is ranked on the countries that fought in it (books.FightersRanking).
ALTER TYPE round_ranking ADD VALUE IF NOT EXISTS 'fighters';
