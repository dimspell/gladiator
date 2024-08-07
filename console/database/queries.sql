-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = ?
LIMIT 1;

-- name: GetUserByName :one
SELECT *
FROM users
WHERE username = ?
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (username, password)
VALUES (?, ?)
RETURNING *;

-- name: ListCharacters :many
SELECT *
FROM characters
WHERE user_id = ?;

-- name: FindCharacter :one
SELECT *
FROM characters
WHERE character_name = ?
  AND user_id = ?;

-- name: CreateCharacter :one
INSERT INTO characters (strength,
                        agility,
                        wisdom,
                        constitution,
                        health_points,
                        magic_points,
                        experience_points,
                        money,
                        score_points,
                        class_type,
                        skin_carnation,
                        hair_style,
                        light_armour_legs,
                        light_armour_torso,
                        light_armour_hands,
                        light_armour_boots,
                        full_armour,
                        armour_emblem,
                        helmet,
                        secondary_weapon,
                        primary_weapon,
                        shield,
                        unknown_equipment_slot,
                        gender,
                        level,
                        edged_weapons,
                        blunted_weapons,
                        archery,
                        polearms,
                        wizardry,
                        holy_magic,
                        dark_magic,
                        bonus_points,
                        character_name,
                        user_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateCharacterStats :exec
UPDATE characters
SET strength               = ?,
    agility                = ?,
    wisdom                 = ?,
    constitution           = ?,
    health_points          = ?,
    magic_points           = ?,
    experience_points      = ?,
    money                  = ?,
    score_points           = ?,
    class_type             = ?,
    skin_carnation         = ?,
    hair_style             = ?,
    light_armour_legs      = ?,
    light_armour_torso     = ?,
    light_armour_hands     = ?,
    light_armour_boots     = ?,
    full_armour            = ?,
    armour_emblem          = ?,
    helmet                 = ?,
    secondary_weapon       = ?,
    primary_weapon         = ?,
    shield                 = ?,
    unknown_equipment_slot = ?,
    gender                 = ?,
    level                  = ?,
    edged_weapons          = ?,
    blunted_weapons        = ?,
    archery                = ?,
    polearms               = ?,
    wizardry               = ?,
    holy_magic             = ?,
    dark_magic             = ?,
    bonus_points           = ?
WHERE character_name = ?
  AND user_id = ?;

-- name: UpdateCharacterSpells :exec
UPDATE characters
SET spells = ?
WHERE character_name = ?
  AND user_id = ?;

-- name: UpdateCharacterInventory :exec
UPDATE characters
SET inventory = ?
WHERE character_name = ?
  AND user_id = ?;

-- name: DeleteCharacter :exec
DELETE
FROM characters
WHERE character_name = ?
  AND user_id = ?;

-- name: SelectRanking :many
SELECT ROW_NUMBER() over (ORDER BY score_points) as position,
       score_points,
       username,
       character_name
FROM characters
         JOIN users ON characters.user_id = users.id
WHERE class_type = ?
ORDER BY score_points
LIMIT 10 OFFSET ?;

-- name: GetCurrentUser :one
SELECT position, cte.score_points, cte.username, cte.character_name
FROM (SELECT ROW_NUMBER() over (ORDER BY score_points) as position,
             score_points,
             username,
             character_name
      FROM characters
               JOIN users ON characters.user_id = users.id
      WHERE users.id = ?
        AND characters.character_name = ?) as cte
LIMIT 1;

-- name: ListGameRooms :many
SELECT *
FROM game_rooms;

-- name: GetGameRoom :one
SELECT id,
       name,
       password,
       host_ip_address,
       map_id,
       created_by,
       host_user_id
FROM game_rooms
WHERE game_rooms.name = ?
LIMIT 1;

-- name: CreateGameRoom :one
INSERT INTO game_rooms (name, password, host_ip_address, map_id, created_by, host_user_id)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetGameRoomPlayers :many
SELECT DISTINCT characters.user_id,
                username,
                character_name,
                class_type,
                ip_address,
                game_rooms.host_ip_address == game_room_players.ip_address as is_host
FROM game_rooms
         JOIN game_room_players ON game_rooms.id = game_room_players.game_room_id
         JOIN characters ON game_room_players.character_id = characters.id
         JOIN users on users.id = characters.user_id
WHERE game_rooms.id = ?
ORDER BY game_rooms.host_ip_address == game_room_players.ip_address DESC,
         game_room_players.added_at ASC;

-- name: ExistPlayerInRoom :one
SELECT 1 as exist
FROM game_room_players
WHERE game_room_id = ?
  AND character_id = ?;

-- name: AddPlayerToRoom :exec
INSERT INTO game_room_players (game_room_id, user_id, character_id, ip_address, added_at)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteAllGameRoomPlayers :exec
DELETE
FROM game_room_players
WHERE TRUE;

-- name: DeleteAllGameRooms :exec
DELETE
FROM game_rooms
WHERE TRUE;


/*
-- -- name: RemovePlayerFromRoom :exec
-- DELETE
-- FROM game_room_players
-- WHERE game_room_id = ?
--   AND character_id = ?;

-- -- name: DeleteGameRoom :exec
-- -- DELETE
-- -- FROM game_rooms
-- -- WHERE game_rooms.id = ?;
*/
