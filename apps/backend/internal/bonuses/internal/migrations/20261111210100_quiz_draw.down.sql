UPDATE rule_sets SET rules = jsonb_set(rules, '{quiz}',
    rules -> 'quiz' - 'target' - 'width' - 'floor' - 'floorAfter' - 'explore')
WHERE rules ? 'quiz';
