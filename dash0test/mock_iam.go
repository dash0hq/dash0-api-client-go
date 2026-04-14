package dash0test

import (
	"context"

	"github.com/dash0hq/dash0-api-client-go/aws"
)

// MockIAMClient is a configurable mock implementation of [aws.IAMOperations].
// Set the function fields to customize behavior for each test.
//
// Example:
//
//	mock := &dash0test.MockIAMClient{
//	    GetCallerAccountIDFunc: func(ctx context.Context) (string, error) {
//	        return "123456789012", nil
//	    },
//	}
type MockIAMClient struct {
	GetCallerAccountIDFunc        func(ctx context.Context) (string, error)
	CreateReadOnlyRoleFunc        func(ctx context.Context, params aws.RoleParams) (*aws.RoleInfo, error)
	CreateInstrumentationRoleFunc func(ctx context.Context, params aws.RoleParams) (*aws.RoleInfo, error)
	ReadRoleFunc                  func(ctx context.Context, roleName string) (*aws.RoleInfo, error)
	DeleteReadOnlyRoleFunc        func(ctx context.Context, roleNamePrefix string) error
	DeleteInstrumentationRoleFunc func(ctx context.Context, roleNamePrefix, accountID string) error
	UpdateRoleTagsFunc            func(ctx context.Context, roleName string, tags map[string]string) error
}

// Verify that MockIAMClient implements aws.IAMOperations.
var _ aws.IAMOperations = (*MockIAMClient)(nil)

func (m *MockIAMClient) GetCallerAccountID(ctx context.Context) (string, error) {
	if m.GetCallerAccountIDFunc != nil {
		return m.GetCallerAccountIDFunc(ctx)
	}
	return "", nil
}

func (m *MockIAMClient) CreateReadOnlyRole(ctx context.Context, params aws.RoleParams) (*aws.RoleInfo, error) {
	if m.CreateReadOnlyRoleFunc != nil {
		return m.CreateReadOnlyRoleFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockIAMClient) CreateInstrumentationRole(ctx context.Context, params aws.RoleParams) (*aws.RoleInfo, error) {
	if m.CreateInstrumentationRoleFunc != nil {
		return m.CreateInstrumentationRoleFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockIAMClient) ReadRole(ctx context.Context, roleName string) (*aws.RoleInfo, error) {
	if m.ReadRoleFunc != nil {
		return m.ReadRoleFunc(ctx, roleName)
	}
	return nil, nil
}

func (m *MockIAMClient) DeleteReadOnlyRole(ctx context.Context, roleNamePrefix string) error {
	if m.DeleteReadOnlyRoleFunc != nil {
		return m.DeleteReadOnlyRoleFunc(ctx, roleNamePrefix)
	}
	return nil
}

func (m *MockIAMClient) DeleteInstrumentationRole(ctx context.Context, roleNamePrefix, accountID string) error {
	if m.DeleteInstrumentationRoleFunc != nil {
		return m.DeleteInstrumentationRoleFunc(ctx, roleNamePrefix, accountID)
	}
	return nil
}

func (m *MockIAMClient) UpdateRoleTags(ctx context.Context, roleName string, tags map[string]string) error {
	if m.UpdateRoleTagsFunc != nil {
		return m.UpdateRoleTagsFunc(ctx, roleName, tags)
	}
	return nil
}
