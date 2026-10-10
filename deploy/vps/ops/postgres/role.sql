SELECT 'CREATE ROLE ops_reader'
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'ops_reader') \gexec

ALTER ROLE ops_reader WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION
    CONNECTION LIMIT 3 PASSWORD :'password';

GRANT pg_read_all_data TO ops_reader;

ALTER ROLE ops_reader SET default_transaction_read_only = on;
ALTER ROLE ops_reader SET statement_timeout = '20s';
