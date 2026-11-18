ALTER TABLE tiles ADD CONSTRAINT tiles_board_fkey FOREIGN KEY (board) REFERENCES boards (id) NOT VALID;
