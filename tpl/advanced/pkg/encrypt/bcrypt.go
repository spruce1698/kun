/**
 * @Author: spruce
 * @Date: 2026-07-06
 * @Desc: bcrypt 密码加密/校验
 */

package encrypt

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// bcryptMaxLen 是 bcrypt 密钥的最大字节数(72)。
// bcrypt 会静默截断超过 72 字节的密码,导致两个前缀相同的长密码被视为相同,
// 因此在加密与校验前统一拒绝超长密码,避免静默截断。
const bcryptMaxLen = 72

// BcryptHash 使用 bcrypt 对明文密码加密,返回密文
func BcryptHash(password string) (string, error) {
	if len(password) > bcryptMaxLen {
		return "", errors.New("password too long: bcrypt accepts at most 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// BcryptCompare 校验明文密码与密文是否匹配
func BcryptCompare(hash, password string) bool {
	if len(password) > bcryptMaxLen {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
