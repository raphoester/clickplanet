-- The battle's tops make room for the season's: season_0_top_N becomes season_0_battle_top_N.
UPDATE titles SET title = replace(title, 'season_0_top_', 'season_0_battle_top_')
WHERE title IN ('season_0_top_1', 'season_0_top_2', 'season_0_top_3');

UPDATE worn_titles SET title = replace(title, 'season_0_top_', 'season_0_battle_top_')
WHERE title IN ('season_0_top_1', 'season_0_top_2', 'season_0_top_3');
