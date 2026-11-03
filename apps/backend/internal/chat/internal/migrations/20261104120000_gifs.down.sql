DELETE FROM messages WHERE seq IN (SELECT message_seq FROM gifs);
DROP TABLE gifs;
