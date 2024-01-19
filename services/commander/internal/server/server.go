package server

import (
	"RocketRankBot/services/commander/internal/bot"
	"RocketRankBot/services/commander/rpc/commander"
	"context"
)

type Server struct {
	botInstance bot.Bot
}

func NewServer(botInstance bot.Bot) Server {
	return Server{botInstance: botInstance}
}

func (s *Server) ExecutePossibleCommand(ctx context.Context, req *commander.ExecutePossibleCommandReq) (*commander.ExecutePossibleCommandRes, error) {
	go s.botInstance.ExecutePossibleCommand(ctx, req)
	return &commander.ExecutePossibleCommandRes{}, nil
}
