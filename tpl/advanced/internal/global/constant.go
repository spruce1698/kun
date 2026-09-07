package global

// Context Key 统一管理
const (
	CtxAuthToken  string = "Auth.Token"
	CtxAuthUserId string = "Auth.UserId" // int64
	CtxAuthRoleId string = "Auth.RoleId" // int64
)

// 路由前缀常量
const (
	RouterPrefixApi = "/api"
	RouterPrefixMgr = "/mgr"
)
