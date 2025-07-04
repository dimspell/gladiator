package console

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/model"
)

var _ multiv1connect.CharacterServiceHandler = (*characterServiceServer)(nil)

type characterServiceServer struct {
	DB *database.SQLite
}

// ListCharacters returns a list of all characters of a user.
func (s *characterServiceServer) ListCharacters(ctx context.Context, req *connect.Request[multiv1.ListCharactersRequest]) (*connect.Response[multiv1.ListCharactersResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	user, err := s.DB.Read.GetUserByID(ctx, req.Msg.UserId)
	if err != nil {
		slog.Warn("could not get user", logging.Error(err), "user_id", req.Msg.GetUserId())
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user not found"))
	}

	characters, err := s.DB.Read.ListCharacters(ctx, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	chars := make([]*multiv1.Character, len(characters))
	for i, character := range characters {
		info := model.CharacterInfo{
			Strength:             uint16(character.Strength),
			Agility:              uint16(character.Agility),
			Wisdom:               uint16(character.Wisdom),
			Constitution:         uint16(character.Constitution),
			HealthPoints:         uint16(character.HealthPoints),
			MagicPoints:          uint16(character.MagicPoints),
			ExperiencePoints:     uint32(character.ExperiencePoints),
			Money:                uint32(character.Money),
			ScorePoints:          uint32(character.ScorePoints),
			ClassType:            model.ClassType(character.ClassType),
			SkinCarnation:        model.SkinCarnation(character.SkinCarnation),
			HairStyle:            model.HairStyle(character.HairStyle),
			LightArmourLegs:      model.EquipmentSlot(character.LightArmourLegs),
			LightArmourTorso:     model.EquipmentSlot(character.LightArmourTorso),
			LightArmourHands:     model.EquipmentSlot(character.LightArmourHands),
			LightArmourBoots:     model.EquipmentSlot(character.LightArmourBoots),
			FullArmour:           model.EquipmentSlot(character.FullArmour),
			ArmourEmblem:         model.EquipmentSlot(character.ArmourEmblem),
			Helmet:               model.EquipmentSlot(character.Helmet),
			SecondaryWeapon:      model.EquipmentSlot(character.SecondaryWeapon),
			PrimaryWeapon:        model.EquipmentSlot(character.PrimaryWeapon),
			Shield:               model.EquipmentSlot(character.Shield),
			UnknownEquipmentSlot: model.EquipmentSlot(character.UnknownEquipmentSlot),
			Gender:               model.Gender(character.Gender),
			Level:                byte(character.Level),
			EdgedWeapons:         uint16(character.EdgedWeapons),
			BluntedWeapons:       uint16(character.BluntedWeapons),
			Archery:              uint16(character.Archery),
			Polearms:             uint16(character.Polearms),
			Wizardry:             uint16(character.Wizardry),
			HolyMagic:            uint16(character.HolyMagic),
			DarkMagic:            uint16(character.DarkMagic),
			BonusPoints:          uint16(character.BonusPoints),
		}

		inventory, _ := base64.StdEncoding.DecodeString(character.Inventory.String)
		spells, _ := base64.StdEncoding.DecodeString(character.Spells.String)

		chars[i] = &multiv1.Character{
			UserId:        user.ID,
			CharacterId:   character.ID,
			CharacterName: character.CharacterName,
			Stats:         info.ToBytes(),
			Inventory:     inventory,
			Spells:        spells,
		}
	}

	resp := connect.NewResponse(&multiv1.ListCharactersResponse{Characters: chars})
	return resp, nil
}

// GetCharacter returns a character by its name and user id.
func (s *characterServiceServer) GetCharacter(ctx context.Context, req *connect.Request[multiv1.GetCharacterRequest]) (*connect.Response[multiv1.GetCharacterResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	character, err := s.DB.Read.FindCharacter(ctx, database.FindCharacterParams{
		UserID:        req.Msg.UserId,
		CharacterName: req.Msg.CharacterName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	info := model.CharacterInfo{
		Strength:             uint16(character.Strength),
		Agility:              uint16(character.Agility),
		Wisdom:               uint16(character.Wisdom),
		Constitution:         uint16(character.Constitution),
		HealthPoints:         uint16(character.HealthPoints),
		MagicPoints:          uint16(character.MagicPoints),
		ExperiencePoints:     uint32(character.ExperiencePoints),
		Money:                uint32(character.Money),
		ScorePoints:          uint32(character.ScorePoints),
		ClassType:            model.ClassType(character.ClassType),
		SkinCarnation:        model.SkinCarnation(character.SkinCarnation),
		HairStyle:            model.HairStyle(character.HairStyle),
		LightArmourLegs:      model.EquipmentSlot(character.LightArmourLegs),
		LightArmourTorso:     model.EquipmentSlot(character.LightArmourTorso),
		LightArmourHands:     model.EquipmentSlot(character.LightArmourHands),
		LightArmourBoots:     model.EquipmentSlot(character.LightArmourBoots),
		FullArmour:           model.EquipmentSlot(character.FullArmour),
		ArmourEmblem:         model.EquipmentSlot(character.ArmourEmblem),
		Helmet:               model.EquipmentSlot(character.Helmet),
		SecondaryWeapon:      model.EquipmentSlot(character.SecondaryWeapon),
		PrimaryWeapon:        model.EquipmentSlot(character.PrimaryWeapon),
		Shield:               model.EquipmentSlot(character.Shield),
		UnknownEquipmentSlot: model.EquipmentSlot(character.UnknownEquipmentSlot),
		Gender:               model.Gender(character.Gender),
		Level:                byte(character.Level),
		EdgedWeapons:         uint16(character.EdgedWeapons),
		BluntedWeapons:       uint16(character.BluntedWeapons),
		Archery:              uint16(character.Archery),
		Polearms:             uint16(character.Polearms),
		Wizardry:             uint16(character.Wizardry),
		HolyMagic:            uint16(character.HolyMagic),
		DarkMagic:            uint16(character.DarkMagic),
		BonusPoints:          uint16(character.BonusPoints),
	}

	inventory, _ := base64.StdEncoding.DecodeString(character.Inventory.String)
	spells, _ := base64.StdEncoding.DecodeString(character.Spells.String)

	resp := connect.NewResponse(&multiv1.GetCharacterResponse{
		Character: &multiv1.Character{
			UserId:        character.UserID,
			CharacterId:   character.ID,
			CharacterName: character.CharacterName,
			Stats:         info.ToBytes(),
			Inventory:     inventory,
			Spells:        spells,
		},
	})
	return resp, nil
}

// CreateCharacter creates a new character for the user.
func (s *characterServiceServer) CreateCharacter(ctx context.Context, req *connect.Request[multiv1.CreateCharacterRequest]) (*connect.Response[multiv1.CreateCharacterResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tx, queries, err := s.DB.WithTx(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	info := model.ParseCharacterInfo(req.Msg.Stats)
	character, err := queries.CreateCharacter(ctx, database.CreateCharacterParams{
		Strength:             int64(info.Strength),
		Agility:              int64(info.Agility),
		Wisdom:               int64(info.Wisdom),
		Constitution:         int64(info.Constitution),
		HealthPoints:         int64(info.HealthPoints),
		MagicPoints:          int64(info.MagicPoints),
		ExperiencePoints:     int64(info.ExperiencePoints),
		Money:                int64(info.Money),
		ScorePoints:          int64(info.ScorePoints),
		ClassType:            int64(info.ClassType),
		SkinCarnation:        int64(info.SkinCarnation),
		HairStyle:            int64(info.HairStyle),
		LightArmourLegs:      int64(info.LightArmourLegs),
		LightArmourTorso:     int64(info.LightArmourTorso),
		LightArmourHands:     int64(info.LightArmourHands),
		LightArmourBoots:     int64(info.LightArmourBoots),
		FullArmour:           int64(info.FullArmour),
		ArmourEmblem:         int64(info.ArmourEmblem),
		Helmet:               int64(info.Helmet),
		SecondaryWeapon:      int64(info.SecondaryWeapon),
		PrimaryWeapon:        int64(info.PrimaryWeapon),
		Shield:               int64(info.Shield),
		UnknownEquipmentSlot: int64(info.UnknownEquipmentSlot),
		Gender:               int64(info.Gender),
		Level:                int64(info.Level),
		EdgedWeapons:         int64(info.EdgedWeapons),
		BluntedWeapons:       int64(info.BluntedWeapons),
		Archery:              int64(info.Archery),
		Polearms:             int64(info.Polearms),
		Wizardry:             int64(info.Wizardry),
		HolyMagic:            int64(info.HolyMagic),
		DarkMagic:            int64(info.DarkMagic),
		BonusPoints:          int64(info.BonusPoints),
		CharacterName:        req.Msg.CharacterName,
		UserID:               req.Msg.UserId,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, errors.Join(err, tx.Rollback()))
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}

	resp := connect.NewResponse(&multiv1.CreateCharacterResponse{
		Character: &multiv1.Character{
			UserId:        req.Msg.UserId,
			CharacterId:   character.ID,
			CharacterName: character.CharacterName,
			Stats:         req.Msg.Stats,
			Inventory:     nil,
			Spells:        nil,
		},
	})
	return resp, nil
}

// PutStats updates the character stats.
func (s *characterServiceServer) PutStats(ctx context.Context, req *connect.Request[multiv1.PutStatsRequest]) (*connect.Response[multiv1.PutStatsResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tx, queries, err := s.DB.WithTx(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	info := model.ParseCharacterInfo(req.Msg.Stats)
	if err := queries.UpdateCharacterStats(ctx, database.UpdateCharacterStatsParams{
		Strength:             int64(info.Strength),
		Agility:              int64(info.Agility),
		Wisdom:               int64(info.Wisdom),
		Constitution:         int64(info.Constitution),
		HealthPoints:         int64(info.HealthPoints),
		MagicPoints:          int64(info.MagicPoints),
		ExperiencePoints:     int64(info.ExperiencePoints),
		Money:                int64(info.Money),
		ScorePoints:          int64(info.ScorePoints),
		ClassType:            int64(info.ClassType),
		SkinCarnation:        int64(info.SkinCarnation),
		HairStyle:            int64(info.HairStyle),
		LightArmourLegs:      int64(info.LightArmourLegs),
		LightArmourTorso:     int64(info.LightArmourTorso),
		LightArmourHands:     int64(info.LightArmourHands),
		LightArmourBoots:     int64(info.LightArmourBoots),
		FullArmour:           int64(info.FullArmour),
		ArmourEmblem:         int64(info.ArmourEmblem),
		Helmet:               int64(info.Helmet),
		SecondaryWeapon:      int64(info.SecondaryWeapon),
		PrimaryWeapon:        int64(info.PrimaryWeapon),
		Shield:               int64(info.Shield),
		UnknownEquipmentSlot: int64(info.UnknownEquipmentSlot),
		Gender:               int64(info.Gender),
		Level:                int64(info.Level),
		EdgedWeapons:         int64(info.EdgedWeapons),
		BluntedWeapons:       int64(info.BluntedWeapons),
		Archery:              int64(info.Archery),
		Polearms:             int64(info.Polearms),
		Wizardry:             int64(info.Wizardry),
		HolyMagic:            int64(info.HolyMagic),
		DarkMagic:            int64(info.DarkMagic),
		BonusPoints:          int64(info.BonusPoints),
		CharacterName:        req.Msg.CharacterName,
		UserID:               req.Msg.UserId,
	}); err != nil {
		return nil, connect.NewError(connect.CodeAborted, errors.Join(err, tx.Rollback()))
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}

	resp := connect.NewResponse(&multiv1.PutStatsResponse{})
	return resp, nil
}

// PutSpells update character spells.
func (s *characterServiceServer) PutSpells(ctx context.Context, req *connect.Request[multiv1.PutSpellsRequest]) (*connect.Response[multiv1.PutSpellsResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	spells := base64.StdEncoding.EncodeToString(req.Msg.Spells)

	tx, queries, err := s.DB.WithTx(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := queries.UpdateCharacterSpells(ctx, database.UpdateCharacterSpellsParams{
		Spells:        sql.NullString{String: spells, Valid: len(req.Msg.Spells) > 0},
		CharacterName: req.Msg.CharacterName,
		UserID:        req.Msg.UserId,
	}); err != nil {
		return nil, connect.NewError(connect.CodeAborted, errors.Join(err, tx.Rollback()))
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}

	resp := connect.NewResponse(&multiv1.PutSpellsResponse{})
	return resp, nil
}

// PutInventoryCharacter update character inventory.
func (s *characterServiceServer) PutInventoryCharacter(ctx context.Context, req *connect.Request[multiv1.PutInventoryRequest]) (*connect.Response[multiv1.PutInventoryResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	inventory := base64.StdEncoding.EncodeToString(req.Msg.Inventory)

	tx, queries, err := s.DB.WithTx(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := queries.UpdateCharacterInventory(ctx, database.UpdateCharacterInventoryParams{
		Inventory:     sql.NullString{String: inventory, Valid: len(req.Msg.Inventory) > 0},
		CharacterName: req.Msg.CharacterName,
		UserID:        req.Msg.UserId,
	}); err != nil {
		return nil, connect.NewError(connect.CodeAborted, errors.Join(err, tx.Rollback()))
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}

	resp := connect.NewResponse(&multiv1.PutInventoryResponse{})
	return resp, nil
}

// DeleteCharacter deletes a character from the database.
func (s *characterServiceServer) DeleteCharacter(ctx context.Context, req *connect.Request[multiv1.DeleteCharacterRequest]) (*connect.Response[multiv1.DeleteCharacterResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tx, queries, err := s.DB.WithTx(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := queries.DeleteCharacter(ctx, database.DeleteCharacterParams{
		CharacterName: req.Msg.CharacterName,
		UserID:        req.Msg.UserId,
	}); err != nil {
		return nil, connect.NewError(connect.CodeAborted, errors.Join(err, tx.Rollback()))
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}

	resp := connect.NewResponse(&multiv1.DeleteCharacterResponse{})
	return resp, nil
}
