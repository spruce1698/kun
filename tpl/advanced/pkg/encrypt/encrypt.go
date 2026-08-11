/**
 * @Author:
 * @Date: 2024-03-28 10:00
 * @Desc: encrypt
 */

package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pbkdf2"
)

/*
0: “sha1”
1: “sha224”
2: “sha256”
3: “sha384”
4: “sha512”
5: “md5”
6: “rmd160”
7: “sha224WithRSAEncryption”
8: “RSA-SHA224”
9: “sha256WithRSAEncryption”
10: “RSA-SHA256”
11: “sha384WithRSAEncryption”
12: “RSA-SHA384”
13: “sha512WithRSAEncryption”
14: “RSA-SHA512”
15: “RSA-SHA1”
16: “ecdsa-with-SHA1”
17: “sha256”
18: “sha224”
19: “sha384”
20: “sha512”
21: “DSA-SHA”
22: “DSA-SHA1”
23: “DSA”
24: “DSA-WITH-SHA224”
25: “DSA-SHA224”
26: “DSA-WITH-SHA256”
27: “DSA-SHA256”
28: “DSA-WITH-SHA384”
29: “DSA-SHA384”
30: “DSA-WITH-SHA512”
31: “DSA-SHA512”
32: “DSA-RIPEMD160”
33: “ripemd160WithRSA”
34: “RSA-RIPEMD160”
35: “md5WithRSAEncryption”
36: “RSA-MD5”
*/

// 对字符串进行MD5加密
func MD5(str string) string {
	hash := md5.New()
	hash.Write([]byte(str))
	return hex.EncodeToString(hash.Sum(nil))
}

// 16位md5
func MD5to16(str string) string {
	return MD5(str)[8:24]
}

// sha256加密
func Sha256(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// sha512加密
func Sha512(data, secret string) string {
	h := hmac.New(sha512.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// Sha1加密
func Sha1(data, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// PBKDF2加密
func PBKDF2(str, salt string, iterations, keySize int) string {
	cipherText := pbkdf2.Key([]byte(str), []byte(salt), iterations, keySize, sha256.New)
	return fmt.Sprintf("%x", md5.Sum(cipherText))
}

// AES
// CBC:密码分组链接模式（Cipher Block Chaining） default
// AES CBC Encrypt — 随机 IV,密文格式: base64(iv + ciphertext)
func AESEncrypt(plainText, encryptKey string, iv ...[]byte) (string, error) {
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	originStr := PKCS5Padding([]byte(plainText), blockSize)

	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}
	blockMode := cipher.NewCBCEncrypter(block, ivValue)
	cipherText := make([]byte, len(originStr))
	blockMode.CryptBlocks(cipherText, originStr)

	// 将 iv 拼接到密文前,解密时可自动提取
	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// AES CBC Decrypt — 自动从密文前提取 iv
func AESDecrypt(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		// 兼容旧格式:显式传入 IV
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("iv length mismatch")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		// 新格式:密文前 blockSize 字节为 iv
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}
	if len(encrypted)%blockSize != 0 {
		return "", errors.New("cipherText is not a multiple of the block size")
	}

	blockModel := cipher.NewCBCDecrypter(block, ivValue)
	originStr := make([]byte, len(encrypted))
	blockModel.CryptBlocks(originStr, encrypted)
	output, err := PKCS5UnPadding(originStr)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ECB:电码本模式（Electronic Codebook Book）ECB is insecure
// AES ECB Encrypt
func AESEncryptECB(plainText, encryptKey string) (string, error) {
	// 创建密码组，长度只能是16、24、32字节
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	// 获取密钥长度
	blockSize := block.BlockSize()
	// 填充
	plainTextByte := PKCS5Padding([]byte(plainText), blockSize)
	cipherText := make([]byte, len(plainTextByte))
	// CEB是把整个明文分成若干段相同的小段，然后对每一小段进行加密
	for bs, be := 0, blockSize; bs < len(plainTextByte); bs, be = bs+blockSize, be+blockSize {
		block.Encrypt(cipherText[bs:be], plainTextByte[bs:be])
	}
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// AES ECB Decrypt
func AESDecryptECB(cipherText, encryptKey string) (string, error) {
	// 创建密码组，长度只能是16、24、32字节
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	// 获取密钥长度
	blockSize := block.BlockSize()
	originStr := make([]byte, len(baseByte))
	for bs, be := 0, blockSize; bs < len(baseByte); bs, be = bs+blockSize, be+blockSize {
		block.Decrypt(originStr[bs:be], baseByte[bs:be])
	}
	// 取消填充
	output, err := PKCS5UnPadding(originStr)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// CFB:密码反馈模式（Cipher FeedBack）
// AES CFB Encrypt encryptKey  长度必须是 16、24、32 其中一个
// 密文格式: base64(iv + ciphertext)
func AESEncryptCFB(plainText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}

	// 补码
	// originStr := PKCS5Padding(plainText, blockSize)
	originStr := []byte(plainText)
	// 加密
	blockMode := cipher.NewCFBEncrypter(block, ivValue)
	cipherText := make([]byte, len(originStr))
	blockMode.XORKeyStream(cipherText, originStr)
	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// AES CFB Decrypt — 自动从密文前提取 iv
func AESDecryptCFB(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("iv length mismatch")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}

	blockModel := cipher.NewCFBDecrypter(block, ivValue)
	originStr := make([]byte, len(encrypted))
	blockModel.XORKeyStream(originStr, encrypted)
	// 取消填充
	// return PKCS5UnPadding(originStr)
	return string(originStr), nil
}

// CTR:计数器模式（Counter）
// AES CTR Encrypt — 密文格式: base64(iv + ciphertext)
func AESEncryptCTR(plainText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	// 创建偏移量iv
	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}

	// CTR 为流模式,明文长度=密文长度,不做块填充
	originStr := []byte(plainText)
	// 加密
	blockMode := cipher.NewCTR(block, ivValue)
	// 定义保存结果变量
	cipherText := make([]byte, len(originStr))
	blockMode.XORKeyStream(cipherText, originStr)
	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// AES CTR Decrypt — 自动从密文前提取 iv
func AESDecryptCTR(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	// 创建偏移量iv
	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("iv length mismatch")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}

	blockModel := cipher.NewCTR(block, ivValue)
	originStr := make([]byte, len(encrypted))
	blockModel.XORKeyStream(originStr, encrypted)
	// CTR 为流模式,无需去填充
	return string(originStr), nil
}

// OFB:输出反馈模式
// AES OFB Encrypt — 密文格式: base64(iv + ciphertext)
func AESEncryptOFB(plainText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	// 创建偏移量iv
	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}

	// OFB 为流模式,明文长度=密文长度,不做块填充
	originStr := []byte(plainText)
	blockMode := cipher.NewOFB(block, ivValue)
	cipherText := make([]byte, len(originStr))
	blockMode.XORKeyStream(cipherText, originStr)
	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// AES OFB Decrypt — 自动从密文前提取 iv
func AESDecryptOFB(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	// 创建偏移量iv
	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("iv length mismatch")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}

	blockModel := cipher.NewOFB(block, ivValue)
	originStr := make([]byte, len(encrypted))
	// 解密
	blockModel.XORKeyStream(originStr, encrypted)
	// OFB 为流模式,无需去填充
	return string(originStr), nil
}

// DES
// CBC:密码分组链接模式（Cipher Block Chaining） default
// DES CBC Encrypt — 密文格式: base64(iv + ciphertext)
func DESEncrypt(plainText, encryptKey string, iv ...[]byte) (string, error) {
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	originStr := PKCS5Padding([]byte(plainText), blockSize)

	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}
	if len(ivValue) < blockSize {
		return "", errors.New("need a multiple of the block size")
	}
	ivValue = ivValue[:blockSize]
	blockMode := cipher.NewCBCEncrypter(block, ivValue)

	cipherText := make([]byte, len(originStr))
	blockMode.CryptBlocks(cipherText, originStr)

	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// DES CBC Decrypt — 自动从密文前提取 iv
func DESDecrypt(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("need a multiple of the block size")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}
	if len(encrypted)%blockSize != 0 {
		return "", errors.New("cipherText is not a multiple of the block size")
	}

	blockModel := cipher.NewCBCDecrypter(block, ivValue)
	originStr := make([]byte, len(encrypted))
	blockModel.CryptBlocks(originStr, encrypted)

	output, err := PKCS5UnPadding(originStr)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ECB:电码本模式（Electronic Codebook Book）ECB is insecure
// DES ECB Encrypt
func DESEncryptECB(plainText, encryptKey string) (string, error) {
	// 创建密码组，长度只能是16、24、32字节
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	// 获取密钥长度
	blockSize := block.BlockSize()
	// 填充
	originStr := PKCS5Padding([]byte(plainText), blockSize)
	cipherText := make([]byte, len(originStr))
	// CEB是把整个明文分成若干段相同的小段，然后对每一小段进行加密
	for bs, be := 0, blockSize; bs < len(originStr); bs, be = bs+blockSize, be+blockSize {
		block.Encrypt(cipherText[bs:be], originStr[bs:be])
	}
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// DES ECB Decrypt
func DESDecryptECB(cipherText, encryptKey string) (string, error) {
	// 创建密码组，长度只能是16、24、32字节
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	// 获取密钥长度
	blockSize := block.BlockSize()
	originStr := make([]byte, len(baseByte))
	for bs, be := 0, blockSize; bs < len(baseByte); bs, be = bs+blockSize, be+blockSize {
		block.Decrypt(originStr[bs:be], baseByte[bs:be])
	}
	// 取消填充
	output, err := PKCS5UnPadding(originStr)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// CFB:密码反馈模式（Cipher FeedBack）
// DES CFB Encrypt — 密文格式: base64(iv + ciphertext)
func DESEncryptCFB(plainText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}
	if len(ivValue) < blockSize {
		return "", errors.New("need a multiple of the block size")
	}
	ivValue = ivValue[:blockSize]

	// 补码
	// originStr := PKCS5Padding(plainText, blockSize)
	originStr := []byte(plainText)
	// 加密
	blockMode := cipher.NewCFBEncrypter(block, ivValue)
	cipherText := make([]byte, len(originStr))
	blockMode.XORKeyStream(cipherText, originStr)
	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// DES CFB Decrypt — 自动从密文前提取 iv
func DESDecryptCFB(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	// 创建偏移量iv
	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("need a multiple of the block size")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}

	blockModel := cipher.NewCFBDecrypter(block, ivValue)
	originStr := make([]byte, len(encrypted))
	blockModel.XORKeyStream(originStr, encrypted)
	// 取消填充
	// return PKCS5UnPadding(originStr)
	return string(originStr), nil
}

// CTR:计数器模式（Counter）
// DES CTR Encrypt — 密文格式: base64(iv + ciphertext)
func DESEncryptCTR(plainText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	// 创建偏移量iv
	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}
	if len(ivValue) < blockSize {
		return "", errors.New("need a multiple of the block size")
	}
	ivValue = ivValue[:blockSize]

	// CTR 为流模式,明文长度=密文长度,不做块填充
	originStr := []byte(plainText)
	// 加密
	blockMode := cipher.NewCTR(block, ivValue)
	// 定义保存结果变量
	cipherText := make([]byte, len(originStr))
	blockMode.XORKeyStream(cipherText, originStr)
	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// DES CTR Decrypt — 自动从密文前提取 iv
func DESDecryptCTR(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	// 创建偏移量iv
	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("need a multiple of the block size")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}

	blockModel := cipher.NewCTR(block, ivValue)
	originStr := make([]byte, len(encrypted))
	blockModel.XORKeyStream(originStr, encrypted)
	// CTR 为流模式,无需去填充
	return string(originStr), nil
}

// OFB:输出反馈模式
// DES OFB Encrypt — 密文格式: base64(iv + ciphertext)
func DESEncryptOFB(plainText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	// 创建偏移量iv
	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}
	if len(ivValue) < blockSize {
		return "", errors.New("need a multiple of the block size")
	}
	ivValue = ivValue[:blockSize]

	// OFB 为流模式,明文长度=密文长度,不做块填充
	originStr := []byte(plainText)
	blockMode := cipher.NewOFB(block, ivValue)
	cipherText := make([]byte, len(originStr))
	blockMode.XORKeyStream(cipherText, originStr)
	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// DES OFB Decrypt — 自动从密文前提取 iv
func DESDecryptOFB(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	// 创建偏移量iv
	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("need a multiple of the block size")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}

	blockModel := cipher.NewOFB(block, ivValue)
	originStr := make([]byte, len(encrypted))
	// 解密
	blockModel.XORKeyStream(originStr, encrypted)
	// OFB 为流模式,无需去填充
	return string(originStr), nil
}

// 3DES
// CBC:密码分组链接模式（Cipher Block Chaining） default
// 3DES CBC Encrypt — 密文格式: base64(iv + ciphertext)
func DES3Encrypt(plainText, encryptKey string, iv ...[]byte) (string, error) {
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	originStr := PKCS5Padding([]byte(plainText), blockSize)

	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}
	if len(ivValue) < blockSize {
		return "", errors.New("need a multiple of the block size")
	}
	ivValue = ivValue[:blockSize]
	blockMode := cipher.NewCBCEncrypter(block, ivValue)

	cipherText := make([]byte, len(originStr))
	blockMode.CryptBlocks(cipherText, originStr)
	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// 3DES CBC Decrypt — 自动从密文前提取 iv
func DES3Decrypt(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("need a multiple of the block size")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}
	if len(encrypted)%blockSize != 0 {
		return "", errors.New("cipherText is not a multiple of the block size")
	}

	blockModel := cipher.NewCBCDecrypter(block, ivValue)
	originStr := make([]byte, len(encrypted))
	blockModel.CryptBlocks(originStr, encrypted)

	output, err := PKCS5UnPadding(originStr)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ECB:电码本模式（Electronic Codebook Book）ECB is insecure
// 3DES ECB Encrypt
func DES3EncryptECB(plainText, encryptKey string) (string, error) {
	// 创建密码组，长度只能是16、24、32字节
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	// 获取密钥长度
	blockSize := block.BlockSize()
	// 填充
	originStr := PKCS5Padding([]byte(plainText), blockSize)
	cipherText := make([]byte, len(originStr))
	// CEB是把整个明文分成若干段相同的小段，然后对每一小段进行加密
	for bs, be := 0, blockSize; bs < len(originStr); bs, be = bs+blockSize, be+blockSize {
		block.Encrypt(cipherText[bs:be], originStr[bs:be])
	}
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// 3DES ECB Decrypt
func DES3DecryptECB(cipherText, encryptKey string) (string, error) {
	// 创建密码组，长度只能是16、24、32字节
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	// 获取密钥长度
	blockSize := block.BlockSize()
	originStr := make([]byte, len(baseByte))
	for bs, be := 0, blockSize; bs < len(baseByte); bs, be = bs+blockSize, be+blockSize {
		block.Decrypt(originStr[bs:be], baseByte[bs:be])
	}
	// 取消填充
	output, err := PKCS5UnPadding(originStr)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// CFB:密码反馈模式（Cipher FeedBack）
// 3DES CFB Encrypt — 密文格式: base64(iv + ciphertext)
func DES3EncryptCFB(plainText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}
	if len(ivValue) < blockSize {
		return "", errors.New("need a multiple of the block size")
	}
	ivValue = ivValue[:blockSize]

	// 补码
	// originStr := PKCS5Padding(plainText, blockSize)
	originStr := []byte(plainText)
	// 加密
	blockMode := cipher.NewCFBEncrypter(block, ivValue)
	cipherText := make([]byte, len(originStr))
	blockMode.XORKeyStream(cipherText, originStr)
	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// 3DES CFB Decrypt — 自动从密文前提取 iv
func DES3DecryptCFB(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}
	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	// 创建偏移量iv
	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("need a multiple of the block size")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}

	blockModel := cipher.NewCFBDecrypter(block, ivValue)
	originStr := make([]byte, len(encrypted))
	blockModel.XORKeyStream(originStr, encrypted)
	return string(originStr), nil
}

// CTR:计数器模式（Counter）
// 3DES CTR Encrypt — 密文格式: base64(iv + ciphertext)
func DES3EncryptCTR(plainText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	originStr := []byte(plainText)

	blockSize := block.BlockSize()
	// 创建偏移量iv
	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}
	if len(ivValue) < blockSize {
		return "", errors.New("need a multiple of the block size")
	}
	ivValue = ivValue[:blockSize]

	// CTR 为流模式,明文长度=密文长度,不做块填充
	// 加密
	blockMode := cipher.NewCTR(block, ivValue)
	// 定义保存结果变量
	cipherText := make([]byte, len(originStr))
	blockMode.XORKeyStream(cipherText, originStr)

	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// 3DES CTR Decrypt — 自动从密文前提取 iv
func DES3DecryptCTR(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	// 创建偏移量iv
	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("need a multiple of the block size")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}

	blockModel := cipher.NewCTR(block, ivValue)
	originStr := make([]byte, len(encrypted))
	blockModel.XORKeyStream(originStr, encrypted)

	// CTR 为流模式,无需去填充
	return string(originStr), nil
}

// OFB:输出反馈模式
// 3DES OFB Encrypt — 密文格式: base64(iv + ciphertext)
func DES3EncryptOFB(plainText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	originStr := []byte(plainText)

	blockSize := block.BlockSize()
	// 创建偏移量iv
	var ivValue []byte
	if len(iv) > 0 {
		ivValue = iv[0]
	} else {
		ivValue = make([]byte, blockSize)
		if _, err := io.ReadFull(rand.Reader, ivValue); err != nil {
			return "", err
		}
	}
	if len(ivValue) < blockSize {
		return "", errors.New("need a multiple of the block size")
	}
	ivValue = ivValue[:blockSize]

	// OFB 为流模式,明文长度=密文长度,不做块填充
	blockMode := cipher.NewOFB(block, ivValue)
	cipherText := make([]byte, len(originStr))
	blockMode.XORKeyStream(cipherText, originStr)

	return base64.StdEncoding.EncodeToString(append(ivValue, cipherText...)), nil
}

// 3DES OFB Decrypt — 自动从密文前提取 iv
func DES3DecryptOFB(cipherText, encryptKey string, iv ...[]byte) (string, error) {
	// 创建block
	block, err := des.NewTripleDESCipher([]byte(encryptKey))
	if err != nil {
		return "", err
	}

	baseByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	if len(baseByte) < blockSize*2 {
		return "", errors.New("cipherText too short")
	}

	// 创建偏移量iv
	var ivValue []byte
	var encrypted []byte
	if len(iv) > 0 {
		ivValue = iv[0]
		if len(ivValue) < blockSize {
			return "", errors.New("need a multiple of the block size")
		}
		ivValue = ivValue[:blockSize]
		encrypted = baseByte
	} else {
		ivValue = baseByte[:blockSize]
		encrypted = baseByte[blockSize:]
	}

	blockModel := cipher.NewOFB(block, ivValue)
	originStr := make([]byte, len(encrypted))
	// 解密
	blockModel.XORKeyStream(originStr, encrypted)

	// OFB 为流模式,无需去填充
	return string(originStr), nil
}

func PKCS5Padding(src []byte, blockSize int) []byte {
	return PKCS7Padding(src, blockSize)
}

func PKCS5UnPadding(src []byte) ([]byte, error) {
	return PKCS7UnPadding(src)
}

func PKCS7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padText...)
}

func PKCS7UnPadding(src []byte) ([]byte, error) {
	length := len(src)
	if length == 0 {
		return src, errors.New("invalid data len")
	}
	// 去掉最后一个字节 unPadding 次
	unPadding := int(src[length-1])
	if unPadding < 1 || unPadding > length {
		return src, errors.New("invalid padding")
	}
	return src[:(length - unPadding)], nil
}

func ZerosPadding(src []byte, blockSize int) []byte {
	paddingCount := blockSize - len(src)%blockSize
	if paddingCount == 0 {
		return src
	} else {
		return append(src, bytes.Repeat([]byte{byte(0)}, paddingCount)...)
	}
}

func ZerosUnPadding(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("empty input")
	}
	for i := len(src) - 1; ; i-- {
		if src[i] != 0 {
			return src[:i+1], nil
		}
		if i == 0 {
			return nil, errors.New("all zeros")
		}
	}
}
