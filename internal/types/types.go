package types

import (
	"errors"

	"github.com/cello-proj/cello/internal/validations"
)

type Target struct {
	Name       string           `json:"name" valid:"required~name is required,alphanumunderscore~name must be alphanumeric underscore,stringlength(4|32)~name must be between 4 and 32 characters"`
	Properties TargetProperties `json:"properties"`
	Type       string           `json:"type" valid:"required~type is required"`
}

// TargetProperties for target
type TargetProperties struct {
	PolicyArns     []string `json:"policy_arns"`
	PolicyDocument string   `json:"policy_document"`
	RoleArn        string   `json:"role_arn" valid:"required~role_arn is required"`
}

// Validate validates Target.
func (target Target) Validate() error {
	v := []func() error{
		func() error { return validations.ValidateStruct(target) },
		func() error {
			if target.Type != "aws_account" {
				return errors.New("type must be one of 'aws_account'")
			}
			return nil
		},
		target.Properties.Validate,
	}

	return validations.Validate(v...)
}

// Validate validates TargetProperties.
func (properties TargetProperties) Validate() error {
	v := []func() error{
		func() error { return validations.ValidateStruct(properties) },
		func() error {
			if !validations.IsValidARN(properties.RoleArn) {
				return errors.New("role_arn must be a valid arn")
			}

			if len(properties.PolicyArns) > 5 {
				return errors.New("policy_arns cannot be more than 5")
			}

			for _, arn := range properties.PolicyArns {
				if !validations.IsValidARN(arn) {
					return errors.New("policy_arns contains an invalid arn")
				}
			}
			return nil
		},
	}

	return validations.Validate(v...)
}

// ProjectToken represents a project token.
type ProjectToken struct {
	ID string `json:"token_id"`
}

// IsEmpty returns whether a struct is empty.
func (p ProjectToken) IsEmpty() bool {
	return p == (ProjectToken{})
}

// Token represents a secrets object/type for a project.
type Token struct {
	CreatedAt    string       `json:"created_at"`
	ExpiresAt    string       `json:"expires_at"`
	ProjectID    string       `json:"project_id"`
	ProjectToken ProjectToken `json:"project_token"`
	RoleID       string       `json:"role_id"`
	Secret       string       `json:"secret"`
}
