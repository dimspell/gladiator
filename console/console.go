package console

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/internal/database"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
)

type Console struct {
	DB      *database.Queries
	Backend *backend.Backend
}

func NewConsole(db *database.Queries, b *backend.Backend) *Console {
	return &Console{
		DB:      db,
		Backend: b,
	}
}

func (c *Console) Serve(ctx context.Context, consoleAddr, backendAddr string) error {
	// "github.com/pocketbase/pocketbase"
	// app := pocketbase.NewWithConfig(pocketbase.Config{
	// 	DefaultDebug:         true,
	// 	DefaultDataDir:       "./pb_data.ignore",
	// 	DefaultEncryptionEnv: "",
	// 	HideStartBanner:      false,
	// 	DataMaxOpenConns:     core.DefaultDataMaxOpenConns,
	// 	DataMaxIdleConns:     core.DefaultDataMaxIdleConns,
	// 	LogsMaxOpenConns:     core.DefaultLogsMaxOpenConns,
	// 	LogsMaxIdleConns:     core.DefaultLogsMaxIdleConns,
	// })
	// return app.Start()

	// consoleServer := server.ConsoleServer{
	// 	DB:      c.DB,
	// 	Backend: c.Backend,
	// }
	// return consoleServer.Serve(ctx, consoleAddr, backendAddr)

	return c.ServeGRPC()
}

func (c *Console) ServeGRPC() error {
	const address = "localhost:8080"

	mux := http.NewServeMux()

	mux.Handle(multiv1connect.NewPetStoreServiceHandler(&petStoreServiceServer{}))
	mux.Handle(multiv1connect.NewCharacterServiceHandler(&characterServiceServer{DB: c.DB}))

	fmt.Println("... Listening on", address)
	return http.ListenAndServe(
		address,
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
}

// petStoreServiceServer implements the PetStoreService API.
type petStoreServiceServer struct {
	multiv1connect.UnimplementedPetStoreServiceHandler
}

// PutPet adds the pet associated with the given request into the PetStore.
func (s *petStoreServiceServer) PutPet(
	ctx context.Context,
	req *connect.Request[multiv1.PutPetRequest],
) (*connect.Response[multiv1.PutPetResponse], error) {
	name := req.Msg.GetName()
	petType := req.Msg.GetPetType()
	log.Printf("Got a request to create a %v named %s", petType, name)

	return connect.NewResponse(&multiv1.PutPetResponse{
		Pet: &multiv1.Pet{
			PetType: multiv1.PetType_PET_TYPE_CAT,
			PetId:   "Id",
			Name:    "Name",
		},
	}), nil
}
