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

// PubKeyEncrypt 公钥加密(OAEP,推荐用于加密场景;明文超长自动分块)
func (s *RSASecurity) PubKeyEncrypt(input string) (string, error) {
	if s.pubKey == nil {
		return "", errors.New(`please set the public key in advance`)
	}
	output := bytes.NewBuffer(nil)
	if err := pubKeyEncryptOAEP(s.pubKey, bytes.NewReader([]byte(input)), output); err != nil {
		return "", err
	}
	binaryByte, err := io.ReadAll(output)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(binaryByte), nil
}

// PriKeyDecrypt 私钥解密(对应 PubKeyEncrypt 的 OAEP 加密)
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
	if err := priKeyDecryptOAEP(s.priKey, bytes.NewReader(base64Input), output); err != nil {
		return "", err
	}
	binaryByte, err := io.ReadAll(output)
	if err != nil {
		return "", err
	}
	return string(binaryByte), nil
}

// PriKeyEncrypt 私钥"加密"——本质是对消息做 SHA-256 + PKCS1v15 签名。
// 注意:这是数字签名用途(认证/防篡改),并非机密性加密;公钥可验证但无法"还原"明文。
// 输入不限长度(签名只对哈希操作),返回 base64 编码的定长签名。
func (s *RSASecurity) PriKeyEncrypt(input string) (string, error) {
	if s.priKey == nil {
		return "", errors.New(`please set the private key in advance`)
	}
	sig, err := priKeySign(s.priKey, []byte(input))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// PubKeyDecrypt 公钥"解密"——历史上对应私钥加密的签名式用法。
// 签名无法还原明文,真正的校验需要原始消息,请改用 Verify(message, signature)。
// 保留方法仅为兼容旧调用签名,直接返回错误提示。
func (s *RSASecurity) PubKeyDecrypt(input string) (string, error) {
	if s.pubKey == nil {
		return "", errors.New(`please set the public key in advance`)
	}
	if _, err := base64.StdEncoding.DecodeString(input); err != nil {
		return "", err
	}
	return "", errors.New("PubKeyDecrypt is deprecated: use Verify(message, signature)")
}

// Verify 用公钥校验消息与签名是否匹配(PriKeyEncrypt 产生的签名)。
func (s *RSASecurity) Verify(message, signature string) error {
	if s.pubKey == nil {
		return errors.New(`please set the public key in advance`)
	}
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}
	return pubKeyVerify(s.pubKey, []byte(message), sig)
}
