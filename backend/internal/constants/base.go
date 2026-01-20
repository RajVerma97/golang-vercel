package constants

type BuildStatus string

const (
	BuildStatusPending  BuildStatus = "pending"
	BuildStatusBuilding BuildStatus = "building"
	BuildStatusFailed   BuildStatus = "failed"
	BuildStatusSuccess  BuildStatus = "success"
)

func (t BuildStatus) String() string {
	return string(t)
}

type DeploymentStatus string

const (
	DeploymentStatusPending DeploymentStatus = "pending"
	DeploymentStatusRunning DeploymentStatus = "running"
	DeploymentStatusStopped DeploymentStatus = "stopped"
	DeploymentStatusFailed  DeploymentStatus = "failed"
)

func (s DeploymentStatus) String() string {
	return string(s)
}
