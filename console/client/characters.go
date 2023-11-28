package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dispel-re/dispel-multi/console/dto"
)

func (c *Client) ListCharacters(ctx context.Context, userId string) ([]dto.Character, error) {
	uri := fmt.Sprintf("%s/user/%s/characters", c.ConsoleAddr, userId)
	characters, err := unmarshalResponse[[]dto.Character](doRequest(ctx, c.HttpClient, http.MethodGet, uri, nil, nil))
	return characters, err
}
