package server

import (
	"RocketRankBot/services/twitchconnector/internal/connector"
	"RocketRankBot/services/twitchconnector/rpc/twitchconnector"
	"context"
	"github.com/rs/zerolog/log"
)

type Server struct {
	connectorInstance connector.Connector
}

func NewServer(connectorInstance connector.Connector) Server {
	return Server{connectorInstance: connectorInstance}
}

func (s *Server) JoinChannel(ctx context.Context, req *twitchconnector.JoinChannelReq) (*twitchconnector.JoinChannelRes, error) {
	if err := s.connectorInstance.JoinChannel(req.TwitchLogin); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("could not execute channel join")
		return nil, err
	}
	return &twitchconnector.JoinChannelRes{}, nil
}

func (s *Server) LeaveChannel(ctx context.Context, req *twitchconnector.LeaveChannelReq) (*twitchconnector.LeaveChannelRes, error) {
	if err := s.connectorInstance.LeaveChannel(req.TwitchLogin); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("could not execute channel leave")
		return nil, err
	}
	return &twitchconnector.LeaveChannelRes{}, nil
}

func (s *Server) SendMessage(ctx context.Context, req *twitchconnector.SendMessageReq) (*twitchconnector.SendMessageRes, error) {
	if err := s.connectorInstance.SendMessage(req.TwitchLogin, req.Message); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("could not execute message send")
		return nil, err
	}
	return &twitchconnector.SendMessageRes{}, nil
}

func (s *Server) SendResponseMessage(ctx context.Context, req *twitchconnector.SendResponseMessageReq) (*twitchconnector.SendResponseMessageRes, error) {
	if err := s.connectorInstance.SendResponseMessage(req.TwitchLogin, req.Message, req.RespondToMessageID); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("could not execute respond message send")
		return nil, err
	}
	return &twitchconnector.SendResponseMessageRes{}, nil
}
