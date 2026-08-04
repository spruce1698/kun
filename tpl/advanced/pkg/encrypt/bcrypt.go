/**
 * @Author: spruce
 * @Date: 2026-07-06
 * @Desc: bcrypt 密码加密/校验
 */

package encrypt

import "golang.org/x/crypto/bcrypt"

// BcryptHash 使用 bcrypt 对明文密码加密,返回密文
func BcryptHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// BcryptCompare 校验明文密码与密文是否匹配
func BcryptCompare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
