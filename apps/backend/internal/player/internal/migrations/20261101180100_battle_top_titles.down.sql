DELETE FROM titles WHERE title IN ('season_0_top_1', 'season_0_top_2', 'season_0_top_3');
DELETE FROM worn_titles WHERE title IN ('season_0_top_1', 'season_0_top_2', 'season_0_top_3');

UPDATE titles SET title = replace(title, 'season_0_battle_top_', 'season_0_top_')
WHERE title IN ('season_0_battle_top_1', 'season_0_battle_top_2', 'season_0_battle_top_3');

UPDATE worn_titles SET title = replace(title, 'season_0_battle_top_', 'season_0_top_')
WHERE title IN ('season_0_battle_top_1', 'season_0_battle_top_2', 'season_0_battle_top_3');
