/**
 * @Author:
 * @Date: 2024-03-28 10:00
 * @Desc: encrypt rsa
 */

package encrypt

import (
	"bytes"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"io"
)

// NewRSASecurity 创建一个新的 RSASecurity 实例，避免使用全局可变状态
func NewRSASecurity() *RSASecurity {
	return &RSASecurity{}
}

type RSASecurity struct {
	pubStr string          // 公钥字符串
	priStr string          // 私钥字符串
	pubKey *rsa.PublicKey  // 公钥
	priKey *rsa.PrivateKey // 私钥
}

// 设置公钥
func (s *RSASecurity) SetPublicKey(pubStr string) (err error) {
	s.pubStr = pubStr
	s.pubKey, err = s.GetPublicKey()
	return err
}

// 设置私钥
func (s *RSASecurity) SetPrivateKey(priStr string) (err error) {
	s.priStr = priStr
	s.priKey, err = s.GetPrivateKey()
	return err
}

// 获得私钥 *rsa.PrivateKey
func (s *RSASecurity) GetPrivateKey() (*rsa.PrivateKey, error) {
	return getPriKey([]byte(s.priStr))
}

// 获得公钥 *rsa.PublicKey
func (s *RSASecurity) GetPublicKey() (*rsa.PublicKey, error) {
	return getPubKey([]byte(s.pubStr))
}

// 公钥加密
func (s *RSASecurity) PubKeyEncrypt(input string) (string, error) {
	if s.pubKey == nil {
		return "", errors.New(`please set the public key in advance`)
	}
	output := bytes.NewBuffer(nil)
	err := pubKeyIO(s.pubKey, bytes.NewReader([]byte(input)), output, true)
	if err != nil {
		return "", err
	}
	binaryByte, err := io.ReadAll(output)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(binaryByte), nil
}

// 公钥解密
func (s *RSASecurity) PubKeyDecrypt(input string) (string, error) {
	if s.pubKey == nil {
		return "", errors.New(`please set the public key in advance`)
	}
	base64Input, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}
	output := bytes.NewBuffer(nil)
	err = pubKeyIO(s.pubKey, bytes.NewReader(base64Input), output, false)
	if err != nil {
		return "", err
	}
	binaryByte, err := io.ReadAll(output)
	if err != nil {
		return "", err
	}
	return string(binaryByte), nil
}

// 私钥加密
func (s *RSASecurity) PriKeyEncrypt(input string) (string, error) {
	if s.priKey == nil {
		return "", errors.New(`please set the private key in advance`)
	}
	output := bytes.NewBuffer(nil)
	err := priKeyIO(s.priKey, bytes.NewReader([]byte(input)), output, true)
	if err != nil {
		return "", err
	}
	binaryByte, err := io.ReadAll(output)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(binaryByte), nil
}

// 私钥解密
func (s *RSASecurity) PriKeyDecrypt(input string) (string, error) {
	if s.priKey == nil {
		return "", errors.New(`please set the private key in advance`)
	}
	if input == "" {
		return "", errors.New(`input is empty`)
	}
	base64Input, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	output := bytes.NewBuffer(nil)
	err = priKeyIO(s.priKey, bytes.NewReader(base64Input), output, false)
	if err != nil {
		return "", err
	}
	binaryByte, err := io.ReadAll(output)
	if err != nil {
		return "", err
	}
	return string(binaryByte), nil
}
