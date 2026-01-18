BEGIN;
CREATE TABLE IF NOT EXISTS RailLine (
    id INTEGER PRIMARY KEY NOT NULL UNIQUE,
    name VARCHAR(16) NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS Station (
    id INTEGER PRIMARY KEY NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS LineStation (
    line_id INTEGER NOT NULL,
    station_id INTEGER NOT NULL,
    FOREIGN KEY (line_id) REFERENCES RailLine(id),
    FOREIGN KEY (station_id) REFERENCES Station(id),
    PRIMARY KEY (line_id, station_id)
);
CREATE TABLE IF NOT EXISTS User (
    id INTEGER PRIMARY KEY NOT NULL UNIQUE,
    -- Maximum length of an email address from https://datatracker.ietf.org/doc/html/rfc5321#section-4.5.3.1
    email VARCHAR(320) NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS UserStation (
    user_id INTEGER NOT NULL,
    station_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES User(id),
    FOREIGN KEY (station_id) REFERENCES Station(id),
    PRIMARY KEY (user_id, station_id)
);
COMMIT;