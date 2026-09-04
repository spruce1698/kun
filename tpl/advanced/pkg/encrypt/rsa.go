/**
 * @Author:
 * @Date: 2024-03-28 10:00
 * @Desc: encrypt rsa
 */

package encrypt

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"hash"
	"io"
)

var (
	ErrDataToLarge = errors.New("message too long for RSA public key size")
	ErrDataLen     = errors.New("data length error")
	ErrPublicKey   = errors.New("get public key error")
	ErrPrivateKey  = errors.New("get private key error")
	ErrSignature   = errors.New("rsa signature verification failed")
)

// rsaHash 签名/验签使用的哈希算法,固定 SHA-256。
const rsaHash = crypto.SHA256

// newHash 返回 rsaHash 对应的 hash.Hash 实例。
func newHash() hash.Hash {
	return sha256.New()
}

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

// ================= RSA 内部加解密与签名辅助实现 =================

// 获取公钥
func getPubKey(publicKey []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("get public key error")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, ErrPublicKey
	}
	return rsaPub, nil
}

// 获取私钥
func getPriKey(privateKey []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("get private key error")
	}
	if pri, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return pri, nil
	}
	pri2, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPri, ok := pri2.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrPrivateKey
	}
	return rsaPri, nil
}

// oaepMaxBlockLen 返回 OAEP 填充下单块可加密的最大明文长度。
func oaepMaxBlockLen(pub *rsa.PublicKey) int {
	k := (pub.N.BitLen() + 7) / 8
	return k - 2*newHash().Size() - 2
}

// keyByteLen 返回密钥字节长度 k(密文块长度固定为 k)。
func keyByteLen(pub *rsa.PublicKey) int {
	return (pub.N.BitLen() + 7) / 8
}

// pubKeyEncryptOAEP 用公钥 OAEP 加密,明文超长时分块加密并拼接。
func pubKeyEncryptOAEP(pub *rsa.PublicKey, in io.Reader, out io.Writer) error {
	chunk := oaepMaxBlockLen(pub)
	if chunk <= 0 {
		return ErrDataToLarge
	}
	buf := make([]byte, chunk)
	for {
		n, err := in.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		block, err := rsa.EncryptOAEP(newHash(), rand.Reader, pub, buf[:n], nil)
		if err != nil {
			return err
		}
		if _, err = out.Write(block); err != nil {
			return err
		}
	}
}

// priKeyDecryptOAEP 用私钥 OAEP 解密,按 k 字节分块。
func priKeyDecryptOAEP(pri *rsa.PrivateKey, in io.Reader, out io.Writer) error {
	k := keyByteLen(&pri.PublicKey)
	buf := make([]byte, k)
	for {
		n, err := io.ReadFull(in, buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if errors.Is(err, io.ErrUnexpectedEOF) {
				// 最后一块不足 k,尝试解密剩余字节
				if n == 0 {
					return nil
				}
				block, derr := rsa.DecryptOAEP(newHash(), rand.Reader, pri, buf[:n], nil)
				if derr != nil {
					return derr
				}
				_, err = out.Write(block)
				return err
			}
			return err
		}
		block, err := rsa.DecryptOAEP(newHash(), rand.Reader, pri, buf[:n], nil)
		if err != nil {
			return err
		}
		if _, err = out.Write(block); err != nil {
			return err
		}
	}
}

// priKeySign 用私钥对消息做 SHA-256 后 PKCS1v15 签名,返回定长签名(k 字节)。
func priKeySign(pri *rsa.PrivateKey, message []byte) ([]byte, error) {
	h := newHash()
	if _, err := h.Write(message); err != nil {
		return nil, err
	}
	return rsa.SignPKCS1v15(rand.Reader, pri, rsaHash, h.Sum(nil))
}

// pubKeyVerify 用公钥校验消息与签名是否匹配。
func pubKeyVerify(pub *rsa.PublicKey, message, signature []byte) error {
	h := newHash()
	if _, err := h.Write(message); err != nil {
		return err
	}
	if err := rsa.VerifyPKCS1v15(pub, rsaHash, h.Sum(nil), signature); err != nil {
		return ErrSignature
	}
	return nil
}
