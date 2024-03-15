CREATE ROLE apps NOLOGIN;
CREATE DATABASE rocketrankbot WITH OWNER apps;

CREATE USER flyway LOGIN PASSWORD 'flyway';
GRANT ALL PRIVILEGES ON DATABASE rocketrankbot TO flyway;

CREATE USER commander LOGIN PASSWORD 'commander';
GRANT apps TO commander;