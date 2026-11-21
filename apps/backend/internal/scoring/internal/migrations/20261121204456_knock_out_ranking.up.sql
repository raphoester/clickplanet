-- A battle that plays the knock-out is ranked on survival first (books.KnockOutRanking).
ALTER TYPE round_ranking ADD VALUE IF NOT EXISTS 'knock_out';
