CREATE TABLE game_rooms
(
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    password        TEXT,
    host_ip_address TEXT    NOT NULL,
    map_id          INTEGER NOT NULL,
    created_by      INTEGER NOT NULL DEFAULT 1,
    host_user_id    INTEGER NOT NULL
);

CREATE TABLE game_room_players
(
    game_room_id INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    ip_address   TEXT    NOT NULL,
    added_at     INTEGER NOT NULL DEFAULT 0,
    user_id      INTEGER NOT NULL,

    PRIMARY KEY (game_room_id, character_id),
    FOREIGN KEY (game_room_id) REFERENCES game_rooms,
    FOREIGN KEY (character_id) REFERENCES characters
);
