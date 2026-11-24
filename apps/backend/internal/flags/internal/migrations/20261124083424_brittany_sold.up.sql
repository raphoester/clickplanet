-- Brittany left the build and is sold, under the code it shipped with, so its land, players, room and points stay its
-- own. Its picture is the build's xb.svg as the server draws an upload (pictures.Redraw).
INSERT INTO flags (id, name, state, decided_at, ends_at,
    picture_hash, picture_width, picture_height, picture_stretch, picture_focus_x, picture_focus_y)
VALUES ('xb', 'Brittany', 'open', now(), now() + interval '30 days',
    'ed47508c67e67af2c0711725441736fbe62b614fb9b9c587a05efc51023c3eb6', 768, 512, false, 0.4344, 0.5151);
