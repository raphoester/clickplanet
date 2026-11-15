UPDATE cards SET
    headline = replace(replace(replace(replace(replace(replace(headline,
        '{boxes.maxBoxesPerHour}', '{bonuses.boxes.maxChargesPerHour}'), '{boxes.offerTtl}', '{bonuses.offerTtl}'),
        '{boxes.missRetry}', '{bonuses.missRetry}'), '{boxes.activeWithin}', '{bonuses.activeWithin}'),
        '{boxes.welcome.', '{bonuses.welcome.'), '{boxes.', '{bonuses.boxes.'),
    line = replace(replace(replace(replace(replace(replace(line,
        '{boxes.maxBoxesPerHour}', '{bonuses.boxes.maxChargesPerHour}'), '{boxes.offerTtl}', '{bonuses.offerTtl}'),
        '{boxes.missRetry}', '{bonuses.missRetry}'), '{boxes.activeWithin}', '{bonuses.activeWithin}'),
        '{boxes.welcome.', '{bonuses.welcome.'), '{boxes.', '{bonuses.boxes.')
WHERE headline LIKE '%{boxes.%' OR line LIKE '%{boxes.%';
