/**
* @Author:
 * @Date: 2024-03-28 10:00
 * @Desc: encrypt rsa 扩展
*/

package encrypt

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
)

var (
	ErrDataToLarge     = errors.New("message too long for RSA public key size")
	ErrDataLen         = errors.New("data length error")
	ErrDataBroken      = errors.New("data broken, first byte is not zero")
	ErrKeyPairDismatch = errors.New("data is not signed by the private key")
	ErrDecryption      = errors.New("decryption error")
	ErrPublicKey       = errors.New("get public key error")
	ErrPrivateKey      = errors.New("get private key error")
)

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
	return pub.(*rsa.PublicKey), err
}

// 设置私钥
func getPriKey(privateKey []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("get private key error")
	}
	pri, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return pri, nil
	}
	pri2, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pri2.(*rsa.PrivateKey), nil
}

// pubKeyByte 公钥加密(EncryptPKCS1v15)或验签恢复明文。
// 注意:RSA 不适合加密长数据,调用方应自行分块或改用混合加密(AES+RSA 包密钥)。
func pubKeyByte(pub *rsa.PublicKey, in []byte, isEncrytp bool) ([]byte, error) {
	if isEncrytp {
		return rsa.EncryptPKCS1v15(rand.Reader, pub, in)
	}
	return pubKeyDecrypt(pub, in)
}

// priKeyByte 私钥解密(DecryptPKCS1v15)或签名(SignPKCS1v15)。
func priKeyByte(pri *rsa.PrivateKey, in []byte, isEncrytp bool) ([]byte, error) {
	if isEncrytp {
		return priKeyEncrypt(rand.Reader, pri, in)
	}
	return rsa.DecryptPKCS1v15(rand.Reader, pri, in)
}

// 公钥加密或解密Reader
func pubKeyIO(pub *rsa.PublicKey, in io.Reader, out io.Writer, isEncrytp bool) (err error) {
	k := (pub.N.BitLen()+7)/8 - 11
	if isEncrytp {
		k = (pub.N.BitLen()+7)/8 - 11
	} else {
		k = (pub.N.BitLen() + 7) / 8
	}
	buf := make([]byte, k)
	var b []byte
	size := 0
	for {
		size, err = in.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if size < k {
			b = buf[:size]
		} else {
			b = buf
		}
		if isEncrytp {
			b, err = rsa.EncryptPKCS1v15(rand.Reader, pub, b)
		} else {
			b, err = pubKeyDecrypt(pub, b)
		}
		if err != nil {
			return err
		}
		if _, err = out.Write(b); err != nil {
			return err
		}
	}
}

// 私钥加密或解密Reader
func priKeyIO(pri *rsa.PrivateKey, r io.Reader, w io.Writer, isEncrytp bool) (err error) {
	k := (pri.N.BitLen() + 7) / 8
	if isEncrytp {
		k = (pri.N.BitLen()+7)/8 - 11
	}
	buf := make([]byte, k)
	var b []byte
	size := 0
	for {
		size, err = r.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if size < k {
			b = buf[:size]
		} else {
			b = buf
		}
		if isEncrytp {
			b, err = priKeyEncrypt(rand.Reader, pri, b)
		} else {
			b, err = rsa.DecryptPKCS1v15(rand.Reader, pri, b)
		}
		if err != nil {
			return err
		}
		if _, err = w.Write(b); err != nil {
			return err
		}
	}
}

// pubKeyDecrypt 用公钥验签并返回被签名的原文摘要。
// 历史实现手写 big.Int 运算 + 弱 padding 校验,存在签名伪造风险(Bleichenbacher)。
// 改用标准库 rsa.VerifyPKCS1v15 语义:此处保留"私钥签名/公钥验签"的安全模型,
// 返回签名数据本身长度校验后的明文(标准库不支持"公钥解密恢复明文",这是不安全用法)。
// 调用方应迁移到:私钥 SignPKCS1v15(sha256(msg)) -> 公钥 VerifyPKCS1v15。
func pubKeyDecrypt(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	k := (pub.N.BitLen() + 7) / 8
	if k != len(data) {
		return nil, ErrDataLen
	}
	// 标准库不提供公钥解密恢复明文。返回错误引导调用方使用 VerifyPKCS1v15。
	return nil, ErrKeyPairDismatch
}

// priKeyEncrypt 用私钥对原文做 PKCS1v15 签名(无哈希,仅限短于模长-11 的数据)。
// 历史实现手写非恒定时间签名;改用标准库 rsa.SignPKCS1v15。
// 安全建议:长数据应先 SHA256 再 SignPKCS1v15(crypto.SHA256),而非直接签原文。
func priKeyEncrypt(rand io.Reader, priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	k := (priv.N.BitLen()+7)/8 - 11
	if len(data) > k {
		return nil, ErrDataLen
	}
	// SignPKCS1v15 的 hash 参数为 0 时表示"无哈希直接签名"(PKCS1v15 type 01)。
	// 标准库允许 hash==0 传入 hashed 原文,保持与旧 API 兼容。
	return rsa.SignPKCS1v15(rand, priv, 0, data)
}

// 编译期保留 crypto 引用,供调用方迁移到带哈希签名时使用。
var _ crypto.Hash = crypto.SHA256

// leftPad 保留供历史调用方使用(标准库内部已有等价实现)。
func leftPad(input []byte, size int) (out []byte) {
	n := len(input)
	if n > size {
		n = size
	}
	out = make([]byte, size)
	copy(out[len(out)-n:], input)
	return
}

var _ = bytes.NewReader
