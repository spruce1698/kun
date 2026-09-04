/**
 * @Author:
 * @Date: 2024-03-28 10:00
 * @Desc: encrypt rsa 扩展
 */

package encrypt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
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

// 设置公钥
func getPubKey(publicKey []byte) (*rsa.PublicKey, error) {
	// decode public key
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("get public key error")
	}
	// x509 parse public key
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

// 设置私钥
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
// k - 2*hash.Size - 2,见 rsa.EncryptOAEP 文档。
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
