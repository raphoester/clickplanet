-- England, Scotland and Wales left the build and are sold, under the codes they shipped with, so their land, players,
-- rooms and points stay their own. Each picture is the build's SVG as the server draws an upload (pictures.Redraw), each
-- pin its capital among Natural Earth's populated places: London, Edinburgh, Cardiff.
INSERT INTO flags (id, name, state, decided_at, ends_at,
    picture_hash, picture_width, picture_height, picture_stretch, picture_focus_x, picture_focus_y,
    pin_latitude, pin_longitude)
VALUES
    ('gb-eng', 'England', 'open', now(), now() + interval '30 days',
        'a5bbe1742db6aa08e5850c063ca9feb11b17c99fbf2546f07ed1f2e57bb44edb', 768, 461, false, 0.5, 0.5,
        51.501941, -0.118668),
    ('gb-sct', 'Scotland', 'open', now(), now() + interval '30 days',
        'e044e22d0d52a24b10503b26b85c66d0c9223e2ddd07842b65d0c9b00ab0a5f8', 768, 461, false, 0.5, 0.5,
        55.948328, -3.219091),
    ('gb-wls', 'Wales', 'open', now(), now() + interval '30 days',
        '625234995a5a9c1da4e98805b1f62053d49ee570cc82c79494ae5e6914fb32d5', 768, 461, false, 0.5, 0.4711,
        51.499995, -3.225008);
