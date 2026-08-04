/**
 * @Author: spruce
 * @Date: 2024-03-28 16:12
 * @Desc: UUid
 */

package utils

import "github.com/google/uuid"

// 生成UUID
func GenUUid() string {
	return uuid.NewString()
}
