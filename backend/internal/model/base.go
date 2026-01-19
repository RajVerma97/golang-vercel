package model

type JobStatus string

const (
	JobStatusPending  JobStatus = "pending"
	JobStatusBuilding JobStatus = "building"
	JobStatusFailed   JobStatus = "failed"
	JobStatusSuccess  JobStatus = "success"
)

func (t JobStatus) String() string {
	return string(t)
}

type Job struct {
	ID      uint64
	RepoUrl string
	Status  JobStatus
}
