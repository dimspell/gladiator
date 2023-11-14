-- -- name: GetAuthor :one
-- SELECT *
-- FROM users
-- WHERE id = ?
-- LIMIT 1;
--
-- -- name: ListAuthors :many
-- SELECT *
-- FROM users
-- ORDER BY username;
--
-- -- name: CreateAuthor :one
-- INSERT INTO users (username, password)
-- VALUES (?, ?)
-- RETURNING *;
--
-- -- name: UpdateAuthor :exec
-- UPDATE users
-- set username = ?,
--     password = ?
-- WHERE id = ?;
--
-- -- name: DeleteAuthor :exec
-- DELETE
-- FROM users
-- WHERE id = ?;

-- name: GetUser :one
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
WHERE user_id = ?
ORDER BY sort_order;

-- name: FindCharacter :one
SELECT *
FROM characters
WHERE user_id = ?
  AND character_name = ?;

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
                        unknown,
                        character_name,
                        user_id,
                        sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
    unknown                = ?
WHERE id = ?;

-- name: UpdateCharacterSpells :exec
UPDATE characters
SET spells = ?
WHERE id = ?;

-- name: UpdateCharacterInventory :exec
UPDATE characters
SET inventory = ?
WHERE id = ?;

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
               JOIN users ON characters.user_id = users.id) as cte
WHERE username = ?
  AND character_name = ?
LIMIT 1;

-- name: ListGameRooms :many
SELECT *
FROM game_rooms;

-- name: GetGameRoom :one
SELECT *
FROM game_rooms
WHERE name = ?
LIMIT 1;

-- name: CreateGameRoom :one
INSERT INTO game_rooms (name, password, host_ip_address)
VALUES (?, ?, ?)
RETURNING *;
