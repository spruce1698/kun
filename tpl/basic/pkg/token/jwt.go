/**
* @Author: spruce
 * @Date: 2024-03-28 10:34
 * @Desc: jwt 签名,验签
*/

package token

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"basic/pkg/encrypt"
	"basic/pkg/utils"
	"basic/pkg/xconfig"
	"basic/pkg/xredis"

	v5 "github.com/golang-jwt/jwt/v5"
)

const (
	ErrorMissingSecret uint32 = 1 << iota
	ErrorExpiredToken
	ErrorAuthorizeElsewhere
	ErrorEmptyAuthHeader
	ErrorMissingIatField
	ErrorMissingExpField
	ErrorInvalidPublicKey
	ErrorInvalidPrivateKey
	ErrorNoPublicKeyFile
	ErrorNoPrivateKeyFile
	ErrorInvalidSigningAlgorithm
	ErrorEmptyToken
	ErrorFailedTokenCreation
	ErrorInvalidToken
	ErrorTokenEmpty // 无效的token

	DefaultQuery = "Authorization"
	DefaultCache = "token:disuse:"

	TokenTypeAccess  = "somebody"
	TokenTypeRefresh = "refresh"

	// SigningAlgorithm 固定的签名算法,Parse 时仅允许该算法,防止算法降级
	SigningAlgorithm = "HS384"

	// minSecretLen HS384 签名可接受的最小密钥长度(字节,即 256bit)。
	minSecretLen = 32
)

var (
	ErrInvalidSigningAlgorithm = newError("invalid signing algorithm", ErrorInvalidSigningAlgorithm)
	ErrInvalidToken            = newError("token is invalid", ErrorInvalidToken)
	ErrEmptyToken              = newError("token is empty", ErrorEmptyToken)
	ErrMissingIatField         = newError("missing iat field", ErrorMissingIatField)
	ErrMissingExpField         = newError("missing exp field", ErrorMissingExpField)
	ErrExpiredToken            = newError("token is expired", ErrorExpiredToken)
)

type (
	Jwt struct {
		Cache         *xredis.Client // 缓存
		ExpireTime    time.Duration  // 过期时间,单位:s
		RefreshTime   time.Duration  // 刷新时间戳,单位:s
		Secret        string         // 加密密钥
		RefreshSecret string         // 刷新密钥
		CSRFKey       string         // CSRF 密钥
		QueryKey      string         // Token查找key
		CacheKey      string         // Token废弃缓存key
	}

	JwtToken struct {
		AccessToken     string // 访问token
		AccessExpireAt  int64  // 访问token过期时间戳
		RefreshToken    string // 刷新token
		RefreshExpireAt int64  // 刷新token过期时间戳
	}
)

// NewJwt 创建
func NewJwt(conf *xconfig.Conf, redisCli *xredis.Client) (*Jwt, error) {
	if redisCli == nil {
		return nil, fmt.Errorf("token 需要redis的支持,请检查redis的配置")
	}
	// 密钥必须由配置提供,禁止回退到硬编码默认值,避免无密钥部署被伪造token
	if conf.Token.Secret == "" || conf.Token.RefreshSecret == "" {
		return nil, fmt.Errorf("token secret 和 refreshSecret 不能为空,请通过配置(或环境变量 TOKEN_SECRET/TOKEN_REFRESHSECRET)提供")
	}
	// HS384 安全要求密钥长度至少 32 字节(256bit),推荐 48 字节(384bit)。
	// 弱密钥会被暴力破解,启动期直接拒绝部署。
	if len(conf.Token.Secret) < minSecretLen || len(conf.Token.RefreshSecret) < minSecretLen {
		return nil, fmt.Errorf("token secret/refreshSecret 长度不足:至少 %d 字节(推荐 48 字节,即 HS384 的 384bit)", minSecretLen)
	}
	j := &Jwt{
		ExpireTime:    2 * time.Hour,
		RefreshTime:   24 * 7 * time.Hour,
		Secret:        conf.Token.Secret,
		RefreshSecret: conf.Token.RefreshSecret,
		CSRFKey:       conf.Token.CSRFKey,
		QueryKey:      DefaultQuery,
		CacheKey:      DefaultCache,
		Cache:         redisCli,
	}

	if conf.Token.QueryKey != "" {
		j.QueryKey = conf.Token.QueryKey
	}
	if conf.Token.CacheKey != "" {
		j.CacheKey = conf.Token.CacheKey
	}
	if conf.Token.Expire > 0 {
		j.ExpireTime = time.Duration(conf.Token.Expire) * time.Second
	}
	if conf.Token.Refresh > 0 {
		j.RefreshTime = time.Duration(conf.Token.Refresh) * time.Second
	}
	if j.RefreshTime <= j.ExpireTime {
		return nil, fmt.Errorf("刷新时间必须大于 token 的过期时间")
	}
	return j, nil
}

// Gen 生成 token AccessToken和RefreshToken,携带 userId 和 roleId
func (j *Jwt) Gen(userId, roleId int64) (*JwtToken, error) {
	cTime := time.Now()

	claims := v5.RegisteredClaims{
		Subject:   strconv.FormatInt(userId, 10),                  // 主题,用户id
		Issuer:    strconv.FormatInt(roleId, 10),                  // 签发人,角色id
		IssuedAt:  v5.NewNumericDate(cTime),                       // 颁发时间
		NotBefore: v5.NewNumericDate(cTime.Add(-1 * time.Second)), // 生效时间
		ExpiresAt: v5.NewNumericDate(cTime.Add(j.ExpireTime)),     // 过期时间
		Audience:  []string{TokenTypeAccess},                      // 观众,[类型]
		ID:        utils.GenStr(0, 16),                            // id
	}
	accessExpireAt := claims.ExpiresAt.Unix()
	accessToken, err := v5.NewWithClaims(v5.SigningMethodHS384, claims).SignedString([]byte(j.Secret))
	if err != nil {
		return nil, err
	}

	refreshClaims := v5.RegisteredClaims{
		Subject:   claims.Subject,                                                                         // 主题,用户id(便于按用户封禁)
		Issuer:    claims.Issuer,                                                                          // 签发人,角色id
		IssuedAt:  claims.IssuedAt,                                                                        // 颁发时间
		NotBefore: claims.NotBefore,                                                                       // 生效时间
		ExpiresAt: v5.NewNumericDate(cTime.Add(j.RefreshTime)),                                            // 过期时间
		Audience:  []string{TokenTypeRefresh},                                                             // 观众,[类型]
		ID:        encrypt.MD5to16(claims.ID + claims.Subject + claims.Issuer + claims.IssuedAt.String()), // 绑定accessToken
	}
	refreshExpireAt := refreshClaims.ExpiresAt.Unix()
	refreshToken, err := v5.NewWithClaims(v5.SigningMethodHS384, refreshClaims).SignedString([]byte(j.RefreshSecret))
	if err != nil {
		return nil, err
	}

	return &JwtToken{
		AccessToken:     accessToken,     // 访问token
		AccessExpireAt:  accessExpireAt,  // 访问token过期时间戳
		RefreshToken:    refreshToken,    // 刷新token
		RefreshExpireAt: refreshExpireAt, // 刷新token过期时间戳
	}, nil
}

// Refresh 刷新token AccessToken和RefreshToken
func (j *Jwt) Refresh(accessToken, refreshToken string) (*JwtToken, error) {
	ctx := context.Background()
	keepOld := true
	// 先判断 refresh token 是否有效
	refreshClaims, err := j.Parse(ctx, refreshToken, TokenTypeRefresh)
	if err != nil {
		return nil, err
	}
	// 3min前的token生成新的token,3min内不变
	if refreshClaims.IssuedAt.Unix() < time.Now().Unix()-180 {
		keepOld = false
	}

	// 解析原的AccessToken,允许已过期(过期时claims仍会返回)
	accessClaims, err := j.Parse(ctx, accessToken, TokenTypeAccess)
	if err != nil && errors.Is(err, ErrExpiredToken) {
		keepOld = false
	} else if err != nil {
		return nil, err
	}
	// 绑定校验:refresh.ID 在 Gen 时由 access 的 ID/Subject/Issuer/IssuedAt 派生,
	// 这里必须用 accessClaims.IssuedAt(而非 refreshClaims.IssuedAt),避免依赖两者恰好相等。
	if encrypt.MD5to16(accessClaims.ID+accessClaims.Subject+accessClaims.Issuer+accessClaims.IssuedAt.String()) != refreshClaims.ID {
		return nil, ErrInvalidToken
	}

	if keepOld {
		return &JwtToken{
			AccessToken:     accessToken,                    // 访问token
			AccessExpireAt:  accessClaims.ExpiresAt.Unix(),  // 访问token过期时间戳
			RefreshToken:    refreshToken,                   // 刷新token
			RefreshExpireAt: refreshClaims.ExpiresAt.Unix(), // 刷新token过期时间戳
		}, nil
	}

	// 原子抢占旧 refreshToken:失败说明已被其它请求轮换过 => 视为复用(泄露信号),
	// 立即吊销该用户的整个 token 家族(access + refresh 都会被按用户黑名单挡掉)
	now := time.Now()
	refreshRemain := refreshClaims.ExpiresAt.Unix() - now.Unix()
	if refreshRemain <= 0 {
		// refresh token 已过期(解析阶段允许过期,到这里才真正判定),直接返回错误,不再尝试抢占
		return nil, ErrExpiredToken
	}
	// 原子抢占旧 refreshToken:失败说明已被其它请求轮换过 => 视为复用(泄露信号),
	// 立即吊销该用户的整个 token 家族(access + refresh 都会被按用户黑名单挡掉)。
	// 注意区分"抢占失败"与"Redis 出错":后者不是复用信号,不能据此吊销整个家族,
	// 否则 Redis 一抖动就会把正常用户全部登出。
	claimed, claimErr := j.DisuseIfAbsent(ctx, refreshToken, refreshRemain)
	if claimErr != nil {
		return nil, fmt.Errorf("刷新token失败: %w", claimErr)
	}
	if !claimed {
		_ = j.Disuse(ctx, refreshClaims.Subject, refreshRemain)
		return nil, ErrInvalidToken
	}
	userId, _ := strconv.ParseInt(accessClaims.Subject, 10, 64)
	roleId, _ := strconv.ParseInt(accessClaims.Issuer, 10, 64)
	token, tokenErr := j.Gen(userId, roleId)
	if tokenErr == nil && token != nil {
		// 旧 accessToken 一并废弃(若仍未过期),刷新后立即失效,避免旧 token 在自然过期前继续可用
		if accessRemain := accessClaims.ExpiresAt.Unix() - now.Unix(); accessRemain > 0 {
			_ = j.Disuse(ctx, accessToken, accessRemain)
		}
	}
	return token, tokenErr
}

// Find 查找
func (j *Jwt) Find(r *http.Request) string {
	// 从HEADER中获取TOKEN(推荐,不会泄漏到日志/Referer)
	if tokenStr := r.Header.Get(j.QueryKey); tokenStr != "" {
		return tokenStr
	}
	// 从COOKIE中获取TOKEN
	if cookie, _ := r.Cookie(j.QueryKey); cookie != nil {
		if cookie.Value != "" {
			return cookie.Value
		}
	}
	// // 从QUERY中获取TOKEN
	// // 注意:token 进 URL 会泄漏到访问日志/Referer/代理日志,默认关闭,确有需要再放开
	// if tokenStr := r.URL.Query().Get(j.QueryKey); tokenStr != "" {
	// 	return tokenStr
	// }
	// // 从FORM中获取TOKEN(同上,默认关闭;且需先 r.ParseForm())
	// if tokenStr := r.Form.Get(j.QueryKey); tokenStr != "" {
	// 	return tokenStr
	// }
	return ""
}

// Parse 解析
// ctx 用于传递 trace/request 信息给 Redis 查询;传 nil 时回退到 background context
func (j *Jwt) Parse(ctx context.Context, tokenStr string, optType ...string) (*v5.RegisteredClaims, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cTime := time.Now()
	tokenStr = strings.TrimSpace(tokenStr)
	if len(tokenStr) >= 7 && strings.EqualFold(tokenStr[:6], "bearer") {
		tokenStr = strings.TrimSpace(tokenStr[6:])
	}
	if tokenStr == "" {
		return nil, ErrEmptyToken
	}
	// tokenStr 在黑名单
	// 必须检查 Redis 错误:忽略它会导致"故障开放"——Redis 不可用时所有已吊销
	// (登出、被封禁、refresh 轮换掉的)token 全部重新生效。宁可拒绝请求也不能放行。
	hasBlacklist, err := j.Cache.Exists(ctx, j.CacheKey+encrypt.MD5(tokenStr)).Result()
	if err != nil {
		return nil, fmt.Errorf("校验token黑名单失败: %w", err)
	}
	if hasBlacklist > 0 {
		return nil, ErrInvalidToken
	}

	tokenType := TokenTypeAccess
	secret := []byte(j.Secret)
	if len(optType) > 0 && optType[0] == TokenTypeRefresh {
		tokenType = TokenTypeRefresh
		secret = []byte(j.RefreshSecret)
	}

	// 解析 token,过期时(jwt.ErrTokenExpired)claims仍会被填充,以便刷新流程使用
	// WithValidMethods 固定签名算法,拒绝 alg=none 及其它算法降级
	// WithExpirationRequired 强制要求 exp 字段:缺失时返回 ErrTokenRequiredClaimErrors,
	// 避免无 exp 的 token 让下方 ExpiresAt(nil) 解引用 panic(远程 DoS)。
	payload := &v5.RegisteredClaims{}
	claim, err := v5.ParseWithClaims(tokenStr, payload, func(token *v5.Token) (any, error) {
		return secret, nil
	}, v5.WithValidMethods([]string{SigningAlgorithm}), v5.WithExpirationRequired())
	if err != nil && !errors.Is(err, v5.ErrTokenExpired) {
		return nil, ErrInvalidToken
	}
	if claim == nil {
		return nil, ErrInvalidToken
	}

	// 判断token类型是否错误
	if len(payload.Audience) != 1 || payload.Audience[0] != tokenType {
		return nil, ErrInvalidToken
	}
	// Subject(用户id)必须存在;对 access/refresh 都检查按用户黑名单,封禁用户即吊销其全部token
	if payload.Subject == "" {
		return nil, ErrInvalidToken
	}
	// Issuer(角色id)必须存在
	if payload.Issuer == "" {
		return nil, ErrInvalidToken
	}
	// 同上,按用户维度的黑名单查询失败也必须拒绝,不能故障开放。
	hasBlacklist, err = j.Cache.Exists(ctx, j.CacheKey+encrypt.MD5(payload.Subject)).Result()
	if err != nil {
		return nil, fmt.Errorf("校验用户黑名单失败: %w", err)
	}
	if hasBlacklist > 0 {
		return nil, ErrInvalidToken
	}

	// 判断token类型是否错误
	if payload.ID == "" {
		return nil, ErrInvalidToken
	}

	// 是否还未生效 生效时间大于过期时间
	// NotBefore 可选,缺失时视作"已生效",跳过校验,避免对 nil *NumericDate 解引用 panic
	if payload.NotBefore != nil && payload.NotBefore.After(cTime) {
		return nil, ErrInvalidToken
	}
	// 是否过期 过期时间小于当前时间,返回claims以便刷新
	// ExpiresAt 由 WithExpirationRequired 保证非 nil;此处再防一次,杜绝 nil 解引用
	if payload.ExpiresAt != nil && payload.ExpiresAt.Before(cTime) {
		return payload, ErrExpiredToken
	}

	return payload, nil
}

// Disuse 废弃的值可以是accountId;也可以是整个key ,expiration 为过期时间
func (j *Jwt) Disuse(ctx context.Context, value string, expiration int64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cacheKey := j.CacheKey + encrypt.MD5(value)
	_, err := j.Cache.SetNX(ctx, cacheKey, 1, time.Duration(expiration)*time.Second).Result()
	return err
}

// DisuseIfAbsent 仅在尚未废弃时废弃,返回是否本次写入(用于 refresh 轮换的原子抢占)
func (j *Jwt) DisuseIfAbsent(ctx context.Context, value string, expiration int64) (bool, error) {
	if expiration <= 0 {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cacheKey := j.CacheKey + encrypt.MD5(value)
	result, err := j.Cache.SetNX(ctx, cacheKey, 1, time.Duration(expiration)*time.Second).Result()
	return result, err
}

// CancelDisuse 取消废弃
func (j *Jwt) CancelDisuse(ctx context.Context, value string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cacheKey := j.CacheKey + encrypt.MD5(value)
	_, err := j.Cache.Del(ctx, cacheKey).Result()
	return err
}
