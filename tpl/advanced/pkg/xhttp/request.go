/**
 * @Author:
 * @Date: 2024-03-28 16:50
 * @Desc: 请求
 */

package xhttp

type (
	PageArg struct {
		OrderField string `form:"orderField"`                                           // 排序字段
		OrderType  int64  `form:"orderType"`                                            // 排序类型 0:升序,1:降序
		Page       int64  `form:"page,default=1" binding:"required,min=1"`              // 添加验证规则
		PageSize   int64  `form:"pageSize,default=20" binding:"required,min=1,max=100"` // 添加验证规则
		LastId     int64  `form:"lastId"`                                               // 上一页最大id
	}
)
