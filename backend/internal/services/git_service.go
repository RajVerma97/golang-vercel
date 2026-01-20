package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"go.uber.org/zap"
)

type GitServiceConfig struct {
}
type GitService struct {
}

func NewGitService(config *GitServiceConfig) *GitService {
	return &GitService{}
}

func (a *GitService) CloneRepository(ctx context.Context, build *dto.Build, tempDirPath string) error {
	// Execute: git clone <repo_url> <temp_dir>
	args := []string{"clone"}

	// If branch is provided, add "-b branchName"
	if build.Branch != nil && *build.Branch != "" {
		args = append(args, "-b", *build.Branch)
	}

	// Add repo URL and destination
	args = append(args, build.RepoUrl, tempDirPath)

	cloneCmd := exec.CommandContext(ctx, "git", args...)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		logger.Error("Git clone failed", err)
		return fmt.Errorf("failed to git clone:%w", err)
	}
	// logger.Debug("Successfully cloned Repository")

	// 2. CHECKOUT COMMIT HASH(if provided)
	if build.CommitHash != nil && *build.CommitHash != "" {
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", *build.CommitHash)
		checkoutCmd.Dir = tempDirPath
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr

		if err := checkoutCmd.Run(); err != nil {
			logger.Error("Git checkout failed", err)
			return fmt.Errorf("failed to checkout commit %s: %w", *build.CommitHash, err)
		}
	}
	logger.Debug("Successfully cloned Repository", zap.String("branch", *build.Branch), zap.String("hash", *build.CommitHash))
	return nil
}
