package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	readOnlyRoleSuffix          = "-read-only"
	instrumentationRoleSuffix   = "-instrumentation"
	instrumentationPolicySuffix = "-lambda-instrumentation"
)

// ReadOnlyRoleName returns the full read-only role name for the given prefix.
func ReadOnlyRoleName(prefix string) string {
	return prefix + readOnlyRoleSuffix
}

// InstrumentationRoleName returns the full instrumentation role name for the given prefix.
func InstrumentationRoleName(prefix string) string {
	return prefix + instrumentationRoleSuffix
}

// InstrumentationPolicyName returns the full instrumentation policy name for the given prefix.
func InstrumentationPolicyName(prefix string) string {
	return prefix + instrumentationPolicySuffix
}

// iamAPI is the subset of the AWS IAM client used by IAMClient.
type iamAPI interface {
	CreateRole(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	DeleteRole(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error)
	AttachRolePolicy(ctx context.Context, params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error)
	DetachRolePolicy(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error)
	PutRolePolicy(ctx context.Context, params *iam.PutRolePolicyInput, optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error)
	DeleteRolePolicy(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error)
	CreatePolicy(ctx context.Context, params *iam.CreatePolicyInput, optFns ...func(*iam.Options)) (*iam.CreatePolicyOutput, error)
	DeletePolicy(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error)
	ListAttachedRolePolicies(ctx context.Context, params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error)
	ListRolePolicies(ctx context.Context, params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error)
	ListRoleTags(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error)
	TagRole(ctx context.Context, params *iam.TagRoleInput, optFns ...func(*iam.Options)) (*iam.TagRoleOutput, error)
	UntagRole(ctx context.Context, params *iam.UntagRoleInput, optFns ...func(*iam.Options)) (*iam.UntagRoleOutput, error)
}

// stsAPI is the subset of the AWS STS client used by IAMClient.
type stsAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// IAMOperations defines the operations available for managing Dash0 IAM roles.
type IAMOperations interface {
	GetCallerAccountID(ctx context.Context) (string, error)
	CreateReadOnlyRole(ctx context.Context, params RoleParams) (*RoleInfo, error)
	CreateInstrumentationRole(ctx context.Context, params RoleParams) (*RoleInfo, error)
	ReadRole(ctx context.Context, roleName string) (*RoleInfo, error)
	DeleteReadOnlyRole(ctx context.Context, roleNamePrefix string) error
	DeleteInstrumentationRole(ctx context.Context, roleNamePrefix, accountID string) error
	UpdateRoleTags(ctx context.Context, roleName string, tags map[string]string) error
}

// IAMClient wraps the AWS IAM and STS clients for role management.
type IAMClient struct {
	iam iamAPI
	sts stsAPI
}

// Verify that IAMClient implements IAMOperations.
var _ IAMOperations = (*IAMClient)(nil)

// RoleParams holds parameters for creating IAM roles.
type RoleParams struct {
	RoleNamePrefix    string
	Dash0AwsAccountID string
	ExternalID        string
	Tags              map[string]string
}

// RoleInfo holds the output from reading or creating a role.
type RoleInfo struct {
	RoleArn  string
	RoleName string
}

// NewIAMClient creates a new AWS IAM client from the given configuration.
// It supports named profile, explicit access keys, or the default credential chain.
func NewIAMClient(ctx context.Context, region, profile, accessKey, secretKey string) (*IAMClient, error) {
	var opts []func(*awsconfig.LoadOptions) error

	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}
	if profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(profile))
	}
	if accessKey != "" && secretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	return &IAMClient{
		iam: iam.NewFromConfig(cfg),
		sts: sts.NewFromConfig(cfg),
	}, nil
}

// GetCallerAccountID returns the AWS account ID of the caller.
func (c *IAMClient) GetCallerAccountID(ctx context.Context) (string, error) {
	output, err := c.sts.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity: %w", err)
	}
	return *output.Account, nil
}

// CreateReadOnlyRole creates the Dash0 read-only IAM role with all required policies.
// The AWS SDK returns [iamtypes.EntityAlreadyExistsException] if the role already exists.
// On failure after partial creation, it attempts best-effort cleanup.
func (c *IAMClient) CreateReadOnlyRole(ctx context.Context, params RoleParams) (*RoleInfo, error) {
	roleName := ReadOnlyRoleName(params.RoleNamePrefix)

	trustPolicy, err := buildTrustPolicy(params.Dash0AwsAccountID, params.ExternalID)
	if err != nil {
		return nil, err
	}

	createOutput, err := c.iam.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Tags:                     convertTags(params.Tags),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create read-only role %q: %w", roleName, err)
	}

	roleArn := *createOutput.Role.Arn

	// Attach ViewOnlyAccess managed policy.
	_, err = c.iam.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(viewOnlyAccessPolicyArn),
	})
	if err != nil {
		c.deleteRoleBestEffort(ctx, roleName)
		return nil, fmt.Errorf("failed to attach ViewOnlyAccess policy to role %q: %w", roleName, err)
	}

	// Attach custom inline policy.
	_, err = c.iam.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String(customPolicyName),
		PolicyDocument: aws.String(readOnlyCustomPolicyJSON),
	})
	if err != nil {
		c.deleteRoleBestEffort(ctx, roleName)
		return nil, fmt.Errorf("failed to put inline policy on role %q: %w", roleName, err)
	}

	return &RoleInfo{
		RoleArn:  roleArn,
		RoleName: roleName,
	}, nil
}

// CreateInstrumentationRole creates the Dash0 instrumentation IAM role.
// The AWS SDK returns [iamtypes.EntityAlreadyExistsException] if the role already exists.
// On failure after partial creation, it attempts best-effort cleanup.
func (c *IAMClient) CreateInstrumentationRole(ctx context.Context, params RoleParams) (*RoleInfo, error) {
	roleName := InstrumentationRoleName(params.RoleNamePrefix)

	trustPolicy, err := buildTrustPolicy(params.Dash0AwsAccountID, params.ExternalID)
	if err != nil {
		return nil, err
	}

	createOutput, err := c.iam.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Tags:                     convertTags(params.Tags),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instrumentation role %q: %w", roleName, err)
	}

	roleArn := *createOutput.Role.Arn

	// Create the managed policy (name is prefix-scoped to avoid collisions).
	policyName := InstrumentationPolicyName(params.RoleNamePrefix)
	policyOutput, err := c.iam.CreatePolicy(ctx, &iam.CreatePolicyInput{
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(instrumentationPolicyJSON),
		Tags:           convertTags(params.Tags),
	})
	if err != nil {
		c.deleteRoleBestEffort(ctx, roleName)
		return nil, fmt.Errorf("failed to create instrumentation policy: %w", err)
	}

	policyArn := *policyOutput.Policy.Arn

	// Attach the policy to the role.
	_, err = c.iam.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		c.deletePolicyBestEffort(ctx, policyArn)
		c.deleteRoleBestEffort(ctx, roleName)
		return nil, fmt.Errorf("failed to attach instrumentation policy to role %q: %w", roleName, err)
	}

	return &RoleInfo{
		RoleArn:  roleArn,
		RoleName: roleName,
	}, nil
}

// ReadRole checks if a role exists and returns its info.
func (c *IAMClient) ReadRole(ctx context.Context, roleName string) (*RoleInfo, error) {
	output, err := c.iam.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get role %q: %w", roleName, err)
	}

	return &RoleInfo{
		RoleArn:  *output.Role.Arn,
		RoleName: roleName,
	}, nil
}

// DeleteReadOnlyRole deletes the read-only role and its attached policies.
// It is idempotent: if the role does not exist, it returns nil.
// If a sub-resource was already removed by a previous partial run, it is skipped.
// Any other error is returned immediately, allowing the caller to retry.
func (c *IAMClient) DeleteReadOnlyRole(ctx context.Context, roleNamePrefix string) error {
	roleName := ReadOnlyRoleName(roleNamePrefix)

	// Verify the role exists before attempting cleanup.
	_, err := c.iam.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to check if read-only role %q exists: %w", roleName, err)
	}

	// Delete inline policy; skip if already removed.
	_, err = c.iam.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(customPolicyName),
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete inline policy from role %q: %w", roleName, err)
	}

	// Detach ViewOnlyAccess managed policy; skip if already detached.
	_, err = c.iam.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(viewOnlyAccessPolicyArn),
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to detach ViewOnlyAccess policy from role %q: %w", roleName, err)
	}

	// Delete the role.
	_, err = c.iam.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete read-only role %q: %w", roleName, err)
	}

	return nil
}

// DeleteInstrumentationRole deletes the instrumentation role and its policy.
// It is idempotent: if the role does not exist, it returns nil.
// If a sub-resource was already removed by a previous partial run, it is skipped.
// Any other error is returned immediately, allowing the caller to retry.
func (c *IAMClient) DeleteInstrumentationRole(ctx context.Context, roleNamePrefix string, accountID string) error {
	roleName := InstrumentationRoleName(roleNamePrefix)
	policyArn := fmt.Sprintf("arn:aws:iam::%s:policy/%s", accountID, InstrumentationPolicyName(roleNamePrefix))

	// Verify the role exists before attempting cleanup.
	_, err := c.iam.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to check if instrumentation role %q exists: %w", roleName, err)
	}

	// Detach policy from role; skip if already detached.
	_, err = c.iam.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to detach instrumentation policy from role %q: %w", roleName, err)
	}

	// Delete the managed policy; skip if already deleted.
	_, err = c.iam.DeletePolicy(ctx, &iam.DeletePolicyInput{
		PolicyArn: aws.String(policyArn),
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete instrumentation policy %q: %w", policyArn, err)
	}

	// Delete the role.
	_, err = c.iam.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete instrumentation role %q: %w", roleName, err)
	}

	return nil
}

// UpdateRoleTags replaces all tags on an IAM role.
func (c *IAMClient) UpdateRoleTags(ctx context.Context, roleName string, tags map[string]string) error {
	existingTags, err := c.iam.ListRoleTags(ctx, &iam.ListRoleTagsInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to list tags for role %q: %w", roleName, err)
	}

	if len(existingTags.Tags) > 0 {
		tagKeys := make([]string, 0, len(existingTags.Tags))
		for _, t := range existingTags.Tags {
			tagKeys = append(tagKeys, *t.Key)
		}
		_, err = c.iam.UntagRole(ctx, &iam.UntagRoleInput{
			RoleName: aws.String(roleName),
			TagKeys:  tagKeys,
		})
		if err != nil {
			return fmt.Errorf("failed to untag role %q: %w", roleName, err)
		}
	}

	if len(tags) > 0 {
		_, err = c.iam.TagRole(ctx, &iam.TagRoleInput{
			RoleName: aws.String(roleName),
			Tags:     convertTags(tags),
		})
		if err != nil {
			return fmt.Errorf("failed to tag role %q: %w", roleName, err)
		}
	}

	return nil
}

// WaitForRolePropagation waits briefly for IAM eventual consistency.
// It respects context cancellation.
func WaitForRolePropagation(ctx context.Context) {
	select {
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
	}
}

// isNotFound returns true if the error is an AWS NoSuchEntityException.
func isNotFound(err error) bool {
	var nse *iamtypes.NoSuchEntityException
	return errors.As(err, &nse)
}

// buildTrustPolicy constructs the IAM trust policy JSON for a Dash0 assume-role.
func buildTrustPolicy(dash0AwsAccountID, externalID string) (string, error) {
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"AWS": dash0AwsAccountID,
				},
				"Action": "sts:AssumeRole",
				"Condition": map[string]interface{}{
					"StringEquals": map[string]interface{}{
						"sts:ExternalId": externalID,
					},
				},
			},
		},
	}
	b, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal trust policy: %w", err)
	}
	return string(b), nil
}

func mustMarshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal static policy JSON: %s", err))
	}
	return string(b)
}

// convertTags converts a map of tags to IAM tag format.
func convertTags(tags map[string]string) []iamtypes.Tag {
	iamTags := make([]iamtypes.Tag, 0, len(tags))
	for k, v := range tags {
		iamTags = append(iamTags, iamtypes.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return iamTags
}

// deleteRoleBestEffort attempts to detach all policies and delete a role.
// Used only during create rollback, where we want to clean up as much as possible.
func (c *IAMClient) deleteRoleBestEffort(ctx context.Context, roleName string) {
	attached, err := c.iam.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, p := range attached.AttachedPolicies {
			_, _ = c.iam.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				RoleName:  aws.String(roleName),
				PolicyArn: p.PolicyArn,
			})
		}
	}

	inline, err := c.iam.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, pName := range inline.PolicyNames {
			_, _ = c.iam.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
				RoleName:   aws.String(roleName),
				PolicyName: aws.String(pName),
			})
		}
	}

	_, _ = c.iam.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
}

// deletePolicyBestEffort attempts to delete a policy.
// Used only during create rollback.
func (c *IAMClient) deletePolicyBestEffort(ctx context.Context, policyArn string) {
	_, _ = c.iam.DeletePolicy(ctx, &iam.DeletePolicyInput{
		PolicyArn: aws.String(policyArn),
	})
}
