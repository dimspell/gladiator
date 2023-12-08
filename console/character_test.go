package console

import (
	"testing"

	"github.com/dispel-re/dispel-multi/console/database"
)

func testDB(t *testing.T) *database.Queries {
	db, err := database.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	queries, err := db.Queries()
	if err != nil {
		t.Fatal(err)
	}
	return queries
}

func TestCharacterServiceServer_ListCharacters(t *testing.T) {
	// db := testDB(t)
	// user, err := db.CreateUser(context.TODO(), database.CreateUserParams{
	// 	Username: "tester",
	// 	Password: "password",
	// })
	// if err != nil {
	// 	t.Error(err)
	// 	t.FailNow()
	// }
	// db.CreateCharacter(context.TODO(), database.CreateCharacterParams{
	// 	CharacterName: "character1",
	// 	UserID:        user.ID,
	// 	SortOrder:     1,
	// })
	// db.CreateCharacter(context.TODO(), database.CreateCharacterParams{
	// 	CharacterName: "character2",
	// 	UserID:        user.ID,
	// 	SortOrder:     2,
	// })
}

func TestCharacterServiceServer_GetCharacter(t *testing.T) {
	// db := testDB(t)
	// user, err := db.CreateUser(context.TODO(), database.CreateUserParams{
	// 	Username: "tester",
	// 	Password: "password",
	// })
	// if err != nil {
	// 	t.Error(err)
	// 	t.FailNow()
	// }
	// db.CreateCharacter(context.TODO(), database.CreateCharacterParams{
	// 	CharacterName: "characterName",
	// 	UserID:        user.ID,
	// 	SortOrder:     1,
	// })
	// db.UpdateCharacterSpells(context.TODO(), database.UpdateCharacterSpellsParams{
	// 	Spells: sql.NullString{
	// 		Valid:  true,
	// 		String: base64.StdEncoding.EncodeToString(spells),
	// 	},
	// 	CharacterName: "characterName",
	// 	UserID:        user.ID,
	// })
}
