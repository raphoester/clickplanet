DELETE FROM knockouts WHERE map = 'world';
ALTER TABLE knockouts ADD CONSTRAINT knockouts_map_check CHECK (map <> 'world');
