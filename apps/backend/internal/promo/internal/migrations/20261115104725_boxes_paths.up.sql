-- The boxes' values left the bonuses module's rules for the boxes module's (boxes.v1.Rules) on 15 November 2026: a card
-- that named one under `bonuses.` names it under `boxes.`, and the hourly cap counts boxes caught.
UPDATE cards SET
    headline = replace(replace(replace(replace(replace(replace(headline,
        '{bonuses.boxes.maxChargesPerHour}', '{boxes.maxBoxesPerHour}'), '{bonuses.boxes.', '{boxes.'),
        '{bonuses.offerTtl}', '{boxes.offerTtl}'), '{bonuses.missRetry}', '{boxes.missRetry}'),
        '{bonuses.activeWithin}', '{boxes.activeWithin}'), '{bonuses.welcome.', '{boxes.welcome.'),
    line = replace(replace(replace(replace(replace(replace(line,
        '{bonuses.boxes.maxChargesPerHour}', '{boxes.maxBoxesPerHour}'), '{bonuses.boxes.', '{boxes.'),
        '{bonuses.offerTtl}', '{boxes.offerTtl}'), '{bonuses.missRetry}', '{boxes.missRetry}'),
        '{bonuses.activeWithin}', '{boxes.activeWithin}'), '{bonuses.welcome.', '{boxes.welcome.')
WHERE headline LIKE '%{bonuses.%' OR line LIKE '%{bonuses.%';
