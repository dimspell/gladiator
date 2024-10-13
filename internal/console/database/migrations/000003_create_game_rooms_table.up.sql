CREATE TABLE game_rooms
(
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    password        TEXT,
    host_ip_address TEXT    NOT NULL,
    map_id          INTEGER NOT NULL
);