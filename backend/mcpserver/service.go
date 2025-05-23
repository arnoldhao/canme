package mcpserver

import (
	"CanMe/backend/consts"
	"CanMe/backend/core/downtasks"
	"CanMe/backend/pkg/logger"
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

type Service struct {
	ctx      context.Context
	svr      *server.MCPServer
	downtask *downtasks.Service
}

func NewService(downtask *downtasks.Service) *Service {
	// Create MCP server
	s := server.NewMCPServer(
		consts.APP_NAME,
		consts.APP_VERSION,
	)
	return &Service{
		svr:      s,
		downtask: downtask,
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.ctx = ctx

	// Add tool
	// download video
	s.svr.AddTool(s.videoDownloader(), s.downloadHandler)
	// download status
	s.svr.AddTool(s.videoDownloaderStatus(), s.downloadStatusHandler)
	// list all tasks
	s.svr.AddTool(s.listDownloadTasks(), s.listTasksHandler)
	// Start the stdio server
	if err := server.ServeStdio(s.svr); err != nil {
		return fmt.Errorf("Server error: %v\n", err)
	}

	sseServer := server.NewSSEServer(s.svr, server.WithBaseURL(fmt.Sprintf("http://localhost:%v", consts.MCP_SERVER_PORT)))
	if err := sseServer.Start(fmt.Sprintf(":%v", consts.MCP_SERVER_PORT)); err != nil {
		logger.Debug("Server error: %v", zap.Error(err))
	}

	return nil
}

func (s *Service) Stop() error {
	return nil
}
