package rbac

// 权限常量
const (
	// 敏感操作权限
	PermissionCreateProject  = "project:create"
	PermissionCreateHabit    = "habit:create"
	PermissionBulkCreateTask = "task:bulk_create"

	// 普通操作权限
	PermissionReadTask   = "task:read"
	PermissionUpdateTask = "task:update"
	PermissionDeleteTask = "task:delete"
)

// 角色常量
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// 角色权限映射
var rolePermissions = map[string][]string{
	RoleUser: {
		PermissionReadTask,
		PermissionUpdateTask,
		PermissionCreateProject,
		PermissionCreateHabit,
		PermissionBulkCreateTask,
	},
	RoleAdmin: {
		PermissionReadTask,
		PermissionUpdateTask,
		PermissionDeleteTask,
		PermissionCreateProject,
		PermissionCreateHabit,
		PermissionBulkCreateTask,
	},
}

// GetUserRole 获取用户角色（目前所有用户都是 user 角色）
// 未来可以从数据库或 JWT token 中获取
func GetUserRole(userID int) string {
	// TODO: 从数据库查询用户角色
	// 目前所有用户都是 user 角色
	return RoleUser
}

// HasPermission 检查用户是否有指定权限
func HasPermission(userID int, permission string) bool {
	role := GetUserRole(userID)
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// CheckPermission 检查用户是否有指定权限（返回错误而不是布尔值，便于处理）
func CheckPermission(userID int, permission string) error {
	if !HasPermission(userID, permission) {
		return &PermissionDeniedError{
			UserID:     userID,
			Permission: permission,
		}
	}
	return nil
}

// PermissionDeniedError 表示权限不足的错误
type PermissionDeniedError struct {
	UserID     int
	Permission string
}

func (e *PermissionDeniedError) Error() string {
	return "insufficient permissions"
}

// ValidateUserIDInPayload 验证 payload 中的 user_id 是否与 token 中的 user_id 匹配
// 这是一个辅助函数，用于在 handler 中验证
func ValidateUserIDInPayload(tokenUserID int, payloadUserID int) error {
	if payloadUserID != tokenUserID {
		return &UserIDMismatchError{
			TokenUserID:   tokenUserID,
			PayloadUserID: payloadUserID,
		}
	}
	return nil
}

// UserIDMismatchError 表示 user_id 不匹配的错误
type UserIDMismatchError struct {
	TokenUserID   int
	PayloadUserID int
}

func (e *UserIDMismatchError) Error() string {
	return "user_id in payload does not match token"
}
