package services

import (
	"context"
	"fmt"
	"os"

	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"go.uber.org/zap"
)

type WorkspaceManagerServiceConfig struct {
}
type WorkspaceManagerService struct {
}

func NewWorkspaceManagerService(config *WorkspaceManagerServiceConfig) *WorkspaceManagerService {
	return &WorkspaceManagerService{}
}

func (s *WorkspaceManagerService) InitializeEnvironment(ctx context.Context, build *dto.Build, tempDirPath string) error {
	// remove existing /tmp/build-%d directory
	if err := s.Cleanup(ctx, tempDirPath); err != nil {
		logger.Error("failed to cleanup container", err)
		return err
	}
	// create fresh directory
	err := os.MkdirAll(tempDirPath, 0755)
	if err != nil {
		logger.Error("Failed to create directory", err, zap.String("tempDirPath", tempDirPath))
		return fmt.Errorf("failed to create directory:%w", err)
	}
	logger.Debug("Successfully created temp dir", zap.String("tempDirPath", tempDirPath))
	return nil
}
func (s *WorkspaceManagerService) Create(ctx context.Context, build *dto.Build, tempDirPath string) error {
	return s.InitializeEnvironment(ctx, build, tempDirPath)
}

func (s *WorkspaceManagerService) Cleanup(ctx context.Context, tempDirPath string) error {
	// remove existing /tmp/build-%d directory
	if _, err := os.Stat(tempDirPath); err == nil {
		logger.Debug("Removing existing temp directory", zap.String("path", tempDirPath))
		err = os.RemoveAll(tempDirPath)
		if err != nil {
			logger.Error("Failed to remove existing temp directory", err)
			return fmt.Errorf("failed to remove existing temp directory;%w", err)
		}
	}
	return nil
}
