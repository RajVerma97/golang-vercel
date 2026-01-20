package dto

import (
	"time"

	"github.com/RajVerma97/golang-vercel/backend/internal/constants"
)

type Container struct {
	ID   string `json:"id" `
	Name string `json:"name"`
	Port string `json:"port"`
}
type Build struct {
	ID            uint64                `json:"id"`
	DeploymentID  uint64                `json:"deployment_id"`
	RepoUrl       string                `json:"repo_url"`
	Branch        string                `json:"branch"`
	CommitHash    string                `json:"commit_hash"`
	Status        constants.BuildStatus `json:"status"`
	Logs          string                `json:"logs"`
	FailureReason *string               `json:"failure_reason"`
	Container     *Container            `json:"container"`
	BinaryPath    *string               `json:"binary_path"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	StartedAt     *time.Time            `json:"started_at"`
	CompletedAt   *time.Time            `json:"completed_at"`
}

type Deployment struct {
	ID        uint64                     `json:"id"`
	BuildID   uint64                     `json:"build_id"`
	URL       string                     `json:"url"`
	Container *Container                 `json:"container"`
	Logs      string                     `json:"logs"`
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
	Status    constants.DeploymentStatus `json:"status"`
	StoppedAt *time.Time                 `json:"stopped_at"`
}
