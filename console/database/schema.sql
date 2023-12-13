CREATE TABLE users
(
    id       INTEGER PRIMARY KEY,
    username TEXT NOT NULL,
    password TEXT NOT NULL
);

CREATE TABLE characters
(
    id                     INTEGER PRIMARY KEY,
    user_id                INTEGER NOT NULL,
    character_name         TEXT    NOT NULL,

    strength               INTEGER NOT NULL,
    agility                INTEGER NOT NULL,
    wisdom                 INTEGER NOT NULL,
    constitution           INTEGER NOT NULL,
    health_points          INTEGER NOT NULL,
    magic_points           INTEGER NOT NULL,
    experience_points      INTEGER NOT NULL,
    money                  INTEGER NOT NULL,
    score_points           INTEGER NOT NULL,
    class_type             INTEGER NOT NULL,
    skin_carnation         INTEGER NOT NULL,
    hair_style             INTEGER NOT NULL,
    light_armour_legs      INTEGER NOT NULL,
    light_armour_torso     INTEGER NOT NULL,
    light_armour_hands     INTEGER NOT NULL,
    light_armour_boots     INTEGER NOT NULL,
    full_armour            INTEGER NOT NULL,
    armour_emblem          INTEGER NOT NULL,
    helmet                 INTEGER NOT NULL,
    secondary_weapon       INTEGER NOT NULL,
    primary_weapon         INTEGER NOT NULL,
    shield                 INTEGER NOT NULL,
    unknown_equipment_slot INTEGER NOT NULL,
    gender                 INTEGER NOT NULL,
    level                  INTEGER NOT NULL,
    edged_weapons          INTEGER NOT NULL,
    blunted_weapons        INTEGER NOT NULL,
    archery                INTEGER NOT NULL,
    polearms               INTEGER NOT NULL,
    wizardry               INTEGER NOT NULL,
    holy_magic             INTEGER NOT NULL,
    dark_magic             INTEGER NOT NULL,
    bonus_points           INTEGER NOT NULL,

    inventory              TEXT,
    spells                 TEXT
);

CREATE TABLE game_rooms
(
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    password        TEXT,
    host_ip_address TEXT    NOT NULL,
    map_id          INTEGER NOT NULL
);

CREATE TABLE game_room_players
(
    game_room_id INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    ip_address   TEXT    NOT NULL
);

