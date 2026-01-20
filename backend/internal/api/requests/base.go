package requests

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/api/errors"
	"github.com/RajVerma97/golang-vercel/backend/internal/api/validation"
)

type DeployRequest struct {
	RepoURL    string  `json:"repo_url" validate:"required"`
	Branch     *string `json:"branch,omitempty"`
	CommitHash *string `json:"commit_hash,omitempty"`
}

func (r *DeployRequest) Validate() error {
	validationErrors := validation.ValidateStruct(r)
	if len(validationErrors) > 0 {
		return errors.NewValidationError(validationErrors)
	}
	return nil
}
