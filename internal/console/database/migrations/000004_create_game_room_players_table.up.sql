CREATE TABLE game_room_players
(
    game_room_id INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    ip_address   TEXT    NOT NULL,

    PRIMARY KEY (game_room_id, character_id),
    FOREIGN KEY (game_room_id) REFERENCES game_rooms,
    FOREIGN KEY (character_id) REFERENCES characters
);
