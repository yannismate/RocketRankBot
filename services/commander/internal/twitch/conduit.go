package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"time"
)

const twitchConduitsURL = "https://api.twitch.tv/helix/eventsub/conduits"

type getConduitsResponse struct {
	Data []struct {
		ID         string `json:"id"`
		ShardCount int    `json:"shard_count"`
	} `json:"data"`
}

func (api *api) getBotConduitID(ctx context.Context) (*string, error) {
	if len(api.botUserID) != 0 {
		return &api.botConduitID, nil
	}

	ids, err := api.getAppConduitIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) > 1 {
		return nil, errors.New(fmt.Sprint("expected 1 or less conduit IDs but got ", len(ids)))
	}

	conduitID := ""

	if len(ids) == 0 {
		log.Ctx(ctx).Info().Msg("No associated conduit found, creating new conduit...")
		resConID, err := api.createAppConduit(ctx, 1)
		if err != nil {
			return nil, err
		}
		conduitID = *resConID
	} else {
		conduitID = ids[0]
	}

	shards, err := api.getAppConduitShards(ctx, conduitID)
	if err != nil {
		return nil, err
	}

	if len(shards.Data) > 1 {
		return nil, errors.New(fmt.Sprint("expected exactly 1 conduit shard but got  ", len(ids)))
	}

	if shards.Data[0].Transport.Callback != api.webHookURL || shards.Data[0].Status != "enabled" {
		log.Ctx(ctx).Info().Msg("Updating conduit shard configuration...")
		_, err := api.updateAppConduitShards(ctx, updateConduitShardsRequest{
			ConduitID: conduitID,
			Shards: []updateConduitShard{
				{
					Transport: updateConduitTransport{
						Method:   "webhook",
						Callback: api.webHookURL,
						Secret:   api.webHookSecret,
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return &conduitID, nil
}

func (api *api) getAppConduitIDs(ctx context.Context) ([]string, error) {
	appToken, err := api.getAppToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", twitchConduitsURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Client-Id", api.clientID)
	req.Header.Set("Authorization", "Bearer "+*appToken)
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	getConduitsRes := getConduitsResponse{}
	err = json.Unmarshal(resData, &getConduitsRes)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(getConduitsRes.Data))
	for _, conduit := range getConduitsRes.Data {
		ids = append(ids, conduit.ID)
	}

	return ids, nil
}

func (api *api) createAppConduit(ctx context.Context, shardCount int) (*string, error) {
	appToken, err := api.getAppToken(ctx)
	if err != nil {
		return nil, err
	}

	body := struct {
		ShardCount int `json:"shard_count"`
	}{shardCount}

	bodyData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", twitchConduitsURL, bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Client-Id", api.clientID)
	req.Header.Set("Authorization", "Bearer "+*appToken)
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	getConduitsRes := getConduitsResponse{}
	err = json.Unmarshal(resData, &getConduitsRes)
	if err != nil {
		return nil, err
	}

	if len(getConduitsRes.Data) != 1 {
		return nil, errors.New(fmt.Sprint("expected 1 conduit IDs but got ", len(getConduitsRes.Data)))
	}

	return &getConduitsRes.Data[0].ID, nil
}

type getConduitShardsResponse struct {
	Data []struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Transport struct {
			Method      string    `json:"method"`
			Callback    string    `json:"callback"`
			SessionID   string    `json:"session_id"`
			ConnectedAt time.Time `json:"connected_at"`
		} `json:"transport"`
	} `json:"data"`
}

// GetAppConduitShards This does not use paginated requests, so the results may be incomplete!
func (api *api) getAppConduitShards(ctx context.Context, conduitId string) (*getConduitShardsResponse, error) {
	appToken, err := api.getAppToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", twitchConduitsURL+"/shards", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Client-Id", api.clientID)
	req.Header.Set("Authorization", "Bearer "+*appToken)
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	getConduitShardsRes := getConduitShardsResponse{}
	err = json.Unmarshal(resData, &getConduitShardsRes)
	if err != nil {
		return nil, err
	}

	return &getConduitShardsRes, nil
}

type updateConduitShardsRequest struct {
	ConduitID string               `json:"conduit_id"`
	Shards    []updateConduitShard `json:"shards"`
}
type updateConduitShard struct {
	Transport updateConduitTransport `json:"transport"`
}

type updateConduitTransport struct {
	Method   string `json:"method"`
	Callback string `json:"callback"`
	Secret   string `json:"secret"`
}

type updateConduitShardsResponse struct {
	Data []struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Transport struct {
			Method      string    `json:"method"`
			Callback    string    `json:"callback"`
			SessionID   string    `json:"session_id"`
			ConnectedAt time.Time `json:"connected_at"`
		} `json:"transport"`
	} `json:"data"`
	Errors []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}
}

func (api *api) updateAppConduitShards(ctx context.Context, updateShardsReq updateConduitShardsRequest) (*updateConduitShardsResponse, error) {
	appToken, err := api.getAppToken(ctx)
	if err != nil {
		return nil, err
	}

	bodyData, err := json.Marshal(updateShardsReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", twitchConduitsURL+"/shards", bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Client-Id", api.clientID)
	req.Header.Set("Authorization", "Bearer "+*appToken)
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	updateConduitsRes := updateConduitShardsResponse{}
	err = json.Unmarshal(resData, &updateConduitsRes)
	if err != nil {
		return nil, err
	}

	if len(updateConduitsRes.Errors) > 0 {
		return nil, errors.New(updateConduitsRes.Errors[0].Message)
	}

	return &updateConduitsRes, nil
}
