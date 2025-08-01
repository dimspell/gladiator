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
