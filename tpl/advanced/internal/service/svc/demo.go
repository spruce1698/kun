package svc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"advanced/internal/event"
	"advanced/internal/global"
	"advanced/internal/repository/cache"
	"advanced/internal/repository/db"
	"advanced/pkg/encrypt"
	"advanced/pkg/token"
	"advanced/pkg/utils"
	"advanced/pkg/xerror"
	"advanced/pkg/xlog"

	"github.com/gorilla/websocket"
	"github.com/jinzhu/copier"
	"github.com/xuri/excelize/v2"
	"golang.org/x/sync/singleflight"
)

//go:generate mockgen -source=./demo.go -destination=../../../test/mocks/service/demo.go  -package mock_service

var _ DemoSvc = (*demoSvc)(nil)

type (
	DemoSvc interface {
		Detail(ctx context.Context, id int64) (*Demo, error)
		FindList(ctx context.Context, args *DemoListArgs) ([]*Demo, int64, error)
		Create(ctx context.Context, args *Demo) error
		Update(ctx context.Context, args *Demo) error
		Delete(ctx context.Context, id int64) error
		SoftDelete(ctx context.Context, id int64) error

		// Login 登录(账号:demo名称,密码:password),返回双token
		Login(ctx context.Context, name, password string) (*token.JwtToken, error)
		// RefreshToken 刷新双token
		RefreshToken(ctx context.Context, accessToken, refreshToken string) (*token.JwtToken, error)
		// Logout 登出,废弃当前token
		Logout(ctx context.Context, tokenStr string) error

		// WSTicket 生成 WebSocket 一次性升级票据(30s 单次有效),规避浏览器 WS 无法设置 Header
		WSTicket(ctx context.Context, userId, roleId int64) (string, error)
		// ConsumeWSTicket 校验并消费票据(单次有效),返回票据绑定的 userId/roleId
		ConsumeWSTicket(ctx context.Context, ticket string) (userId, roleId int64, err error)

		SendMsg(ctx context.Context) error
		AddMsg(ctx context.Context) error
		DelMsg(ctx context.Context) error

		ExportStream(ctx context.Context, handler func(*Demo) error) error
		SSE(ctx context.Context, handler func(*SSEMessage) error) error
		WS(ctx context.Context, conn *websocket.Conn) error
		ExportExcel(ctx context.Context, writer io.Writer) error
	}
	DemoCtx struct {
		*Ctx

		DemoDb    db.DemoDb
		DemoCache cache.DemoCache
		MsgPub    *event.Pub
		Jwt       *token.Jwt
	}
	demoSvc struct {
		ctx *DemoCtx
		sf  singleflight.Group
	}

	DemoListArgs struct {
		OrderField string // 排序字段
		OrderType  int64  // 排序类型 0:降序(默认),1:升序
		Page       int64
		PageSize   int64
		// 游标分页:上一页最后一条记录的 id。
		// 必须从 Handler 透传进来(见 handler/demo/demo.go 的 c.Query("lastId")),否则
		// repo 层的游标分页分支(args.LastId > 0)永远进不去,深分页优化形同虚设。
		LastId int64

		Id   *int64
		Name string
	}

	Demo struct {
		Id       int64
		Name     string
		Test1    float64
		Test4    int32
		RoleId   int64  `json:"roleId"` // 角色id
		Password string `json:"-"`      // 密码(密文),不输出
	}

	SSEMessage struct {
		Time string `json:"time"`
		Msg  string `json:"msg"`
	}
)

func NewDemoSvc(ctx *DemoCtx) DemoSvc {
	return &demoSvc{
		ctx: ctx,
	}
}

// 查找一个(接入 singleflight 防缓存击穿)
func (d *demoSvc) Detail(ctx context.Context, id int64) (*Demo, error) {
	if id <= 0 {
		return nil, xerror.NewError(xerror.InvalidArgument, "Get Demo Detail invalid id", nil)
	}
	xlog.Info(ctx, "Demo Detail")

	// 1. 优先查缓存 (LocalCache + Redis)
	demo, cacheErr := d.ctx.DemoCache.Get(ctx, id)
	if cacheErr == nil {
		result := &Demo{}
		_ = copier.Copy(result, demo)
		return result, nil
	}

	// 2. 缓存未命中: singleflight 合并同 ID 并发请求,防缓存击穿
	val, err, _ := d.sf.Do(fmt.Sprintf("demo:detail:%d", id), func() (any, error) {
		// double check 缓存,避免并发排队协程重复打库
		if dCached, cErr := d.ctx.DemoCache.Get(ctx, id); cErr == nil {
			return dCached, nil
		}

		demoDb, dbErr := d.ctx.DemoDb.Find(ctx, id)
		if dbErr != nil {
			return nil, dbErr
		}

		cacheData := &cache.Demo{
			Id:     demoDb.Id,
			Name:   demoDb.Name,
			Test1:  demoDb.Test1,
			Test4:  demoDb.Test4,
			RoleId: demoDb.RoleId,
		}
		_ = d.ctx.DemoCache.Set(ctx, id, cacheData, 0)
		return cacheData, nil
	})

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, xerror.NewError(xerror.BusinessError, "没有相关记录", err)
		}
		return nil, xerror.NewError(xerror.BusinessError, "Demo Detail 失败", err)
	}

	result := &Demo{}
	_ = copier.Copy(result, val)
	return result, nil
}

// 查找列表
func (d *demoSvc) FindList(ctx context.Context, args *DemoListArgs) ([]*Demo, int64, error) {

	dbArgs := &db.DemoSearch{}
	_ = copier.Copy(dbArgs, args)

	list, total, listErr := d.ctx.DemoDb.ListWithTotal(ctx, dbArgs)
	if listErr != nil {
		if errors.Is(listErr, db.ErrNotFound) {
			return []*Demo{}, 0, nil
		}
		return nil, 0, xerror.NewError(xerror.BusinessError, "Get Demo List 失败", listErr)
	}
	result := make([]*Demo, 0, len(list))
	for _, demo := range list {
		result = append(result, &Demo{
			Id:     demo.Id,
			Name:   demo.Name,
			Test1:  demo.Test1,
			Test4:  demo.Test4,
			RoleId: demo.RoleId,
		})
	}
	return result, total, nil
}

// 创建
func (d *demoSvc) Create(ctx context.Context, args *Demo) error {
	password := args.Password
	if password == "" {
		return xerror.NewError(xerror.InvalidArgument, "密码不能为空", nil)
	}
	hash, hashErr := encrypt.BcryptHash(password)
	if hashErr != nil {
		return xerror.NewError(xerror.BusinessError, "Create Demo 密码加密失败", hashErr)
	}
	_, dbErr := d.ctx.DemoDb.Insert(ctx, &db.Demo{
		Name:     args.Name,
		Test1:    args.Test1,
		Test4:    args.Test4,
		RoleId:   args.RoleId,
		Password: hash,
	})
	if dbErr != nil {
		return xerror.NewError(xerror.BusinessError, "Create Demo 失败", dbErr)
	}
	return nil
}

// 修改(事务示例)
func (d *demoSvc) Update(ctx context.Context, args *Demo) error {
	if args.Id > 0 {
		// 两种事务方式
		// 方式1:
		//err := d.ctx.Conn.Tx(ctx, func(ctx context.Context) error {
		//    preOneId := args.Id - 1
		//    demo, err := d.ctx.DemoDb.Find(ctx, preOneId)
		//    if err != nil {
		//        return err
		//    }
		//    err = d.ctx.DemoDb.Delete(ctx, []int64{preOneId})
		//    if err != nil {
		//        return err
		//    }
		//    if demo.Id == 3 {
		//        return errors.New("id是3,不能修改")
		//    }
		//    _, err = d.ctx.DemoDb.UpdateFields(ctx, args.Id, map[string]any{"name": args.Name, "test1": args.Test1, "test4": args.Test4})
		//    return err
		//})
		// 方式2:
		err := d.ctx.DemoDb.DemoUpdateTrans(ctx, args.Id, &db.Demo{
			Name:  args.Name,
			Test1: args.Test1,
			Test4: args.Test4,
		})

		if err != nil {
			return xerror.NewError(xerror.BusinessError, "Update Demo 失败", err)
		}
		// DemoUpdateTrans 内部会删除 id-1 的记录,故 args.Id 与 args.Id-1 的缓存都要失效。
		d.invalidateCache(ctx, args.Id)
		d.invalidateCache(ctx, args.Id-1)
		return nil
	}
	return xerror.NewError(xerror.BusinessError, "Update Demo 失败", nil)
}

// 物理删除
func (d *demoSvc) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return xerror.NewError(xerror.InvalidArgument, "Delete Demo invalid id", nil)
	}
	err := d.ctx.DemoDb.Delete(ctx, []int64{id})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "Delete Demo 失败", err)
	}
	d.invalidateCache(ctx, id)
	return nil
}

// 逻辑删除
func (d *demoSvc) SoftDelete(ctx context.Context, id int64) error {
	if id <= 0 {
		return xerror.NewError(xerror.InvalidArgument, "SoftDelete Demo invalid id", nil)
	}
	err := d.ctx.DemoDb.SoftDelete(ctx, []int64{id})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "SoftDelete Demo 失败", err)
	}
	d.invalidateCache(ctx, id)
	return nil
}

// invalidateCache 失效指定 id 的缓存(Redis + 本地)。
// 缓存删除失败不阻断主流程(数据已落库),仅记录日志,等 TTL 自然过期。
func (d *demoSvc) invalidateCache(ctx context.Context, id int64) {
	if id <= 0 {
		return
	}
	if err := d.ctx.DemoCache.Delete(ctx, id); err != nil {
		xlog.Error(ctx, fmt.Sprintf("invalidate demo cache failed, id=%d", id), err)
	}
}

// Login 登录(账号:demo名称,密码:password),校验通过后签发双token
func (d *demoSvc) Login(ctx context.Context, name, password string) (*token.JwtToken, error) {
	if name == "" || password == "" {
		return nil, xerror.NewError(xerror.AccountOrPasswordError, "账号或密码不能为空", nil)
	}
	demoDb, err := d.ctx.DemoDb.FindByName(ctx, name)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, xerror.NewError(xerror.AccountOrPasswordError, "账号或密码错误", nil)
		}
		return nil, xerror.NewError(xerror.BusinessError, "Login 失败", err)
	}
	if !encrypt.BcryptCompare(demoDb.Password, password) {
		return nil, xerror.NewError(xerror.AccountOrPasswordError, "账号或密码错误", nil)
	}
	tokenInfo, err := d.ctx.Jwt.Gen(demoDb.Id, demoDb.RoleId)
	if err != nil {
		return nil, xerror.NewError(xerror.BusinessError, "Login 生成token失败", err)
	}
	return tokenInfo, nil
}

// RefreshToken 刷新双token
func (d *demoSvc) RefreshToken(ctx context.Context, accessToken, refreshToken string) (*token.JwtToken, error) {
	tokenInfo, err := d.ctx.Jwt.Refresh(accessToken, refreshToken)
	if err != nil {
		return nil, xerror.NewError(xerror.AuthRefreshErr, "刷新token失败", err)
	}
	return tokenInfo, nil
}

// Logout 登出,将当前token加入废弃黑名单
func (d *demoSvc) Logout(ctx context.Context, tokenStr string) error {
	if tokenStr == "" {
		return nil
	}
	payload, err := d.ctx.Jwt.Parse(ctx, tokenStr)
	if err != nil && !errors.Is(err, token.ErrExpiredToken) {
		return xerror.NewError(xerror.AuthError, "token无效", err)
	}
	if payload == nil {
		return nil
	}
	expiration := payload.ExpiresAt.Unix() - time.Now().Unix()
	if expiration > 0 {
		_ = d.ctx.Jwt.Disuse(ctx, tokenStr, expiration)
	}
	return nil
}

// WSTicket 生成 WebSocket 一次性升级票据(30s,单次有效)
// 浏览器 WebSocket 升级请求无法设置自定义 Header,改用短票据换连接,长 token 不进 URL
func (d *demoSvc) WSTicket(ctx context.Context, userId, roleId int64) (string, error) {
	if userId <= 0 {
		return "", xerror.NewError(xerror.Unauthorized, "用户未登录", nil)
	}
	ticket := utils.GenUUid()
	if err := d.ctx.DemoCache.SetWSTicket(ctx, ticket, userId, roleId, 30*time.Second); err != nil {
		return "", xerror.NewError(xerror.BusinessError, "生成票据失败", err)
	}
	return ticket, nil
}

// ConsumeWSTicket 校验并消费票据(GetDel 原子取删,保证单次有效)
func (d *demoSvc) ConsumeWSTicket(ctx context.Context, ticket string) (int64, int64, error) {
	if ticket == "" {
		return 0, 0, xerror.NewError(xerror.Unauthorized, "票据不能为空", nil)
	}
	userId, roleId, err := d.ctx.DemoCache.ConsumeWSTicket(ctx, ticket)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return 0, 0, xerror.NewError(xerror.Unauthorized, "票据无效或已过期", nil)
		}
		return 0, 0, xerror.NewError(xerror.BusinessError, "票据校验失败", err)
	}
	return userId, roleId, nil
}

// 发送消息
func (d *demoSvc) SendMsg(ctx context.Context) error {
	demoMsg0, err := json.Marshal(&global.PayRecharge{
		TradeNo:    "O100",
		PayType:    global.PayTypeAlipay,
		Status:     global.PayStatusIng,
		UserId:     123,
		Cash:       utils.Yuan2Cent(1.20),
		CreateTime: utils.TimeStr(time.Now()),
		Remark:     "Kafka-无type-消息",
	})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "序列化 Kafka 消息失败", err)
	}
	if err = d.ctx.MsgPub.KafkaCtx(ctx, global.EventTopicPay, demoMsg0); err != nil {
		return err
	}

	demoMsg1, err := json.Marshal(&global.PayRecharge{
		TradeNo:    "O101",
		PayType:    global.PayTypeAlipay,
		Status:     global.PayStatusIng,
		UserId:     123,
		Cash:       utils.Yuan2Cent(1.30),
		CreateTime: utils.TimeStr(time.Now()),
		Remark:     "Kafka-type-消息",
	})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "序列化 Kafka-type 消息失败", err)
	}
	if err = d.ctx.MsgPub.KafkaWithTypeCtx(ctx, global.EventTopicPay, global.EventTypePayRecharge, demoMsg1); err != nil {
		return err
	}

	demoMsg2, err := json.Marshal(&global.PayRecharge{
		TradeNo:    "O102",
		PayType:    global.PayTypeWxpay,
		Status:     global.PayStatusSuccess,
		UserId:     123,
		Cash:       utils.Yuan2Cent(1.50),
		CreateTime: utils.TimeStr(time.Now()),
		Remark:     "5s延迟消息",
	})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "序列化延迟消息失败", err)
	}
	if err = d.ctx.MsgPub.DelayCtx(ctx, global.EventTopicPay, string(demoMsg2), 5); err != nil {
		return err
	}

	demoMsg3, err := json.Marshal(&global.PayRecharge{
		TradeNo:    "O103",
		PayType:    global.PayTypeWxpay,
		Status:     global.PayStatusSuccess,
		UserId:     123,
		Cash:       utils.Yuan2Cent(1.60),
		CreateTime: utils.TimeStr(time.Now()),
		Remark:     "AQ-异步-消息",
	})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "序列化 AQ 异步消息失败", err)
	}
	if err = d.ctx.MsgPub.SyncCtx(ctx, global.EventTopicPay, string(demoMsg3)); err != nil {
		return err
	}

	demoMsg4, err := json.Marshal(&global.PayRecharge{
		TradeNo:    "O104",
		PayType:    global.PayTypeWxpay,
		Status:     global.PayStatusSuccess,
		UserId:     123,
		Cash:       utils.Yuan2Cent(1.70),
		CreateTime: utils.TimeStr(time.Now()),
		Remark:     "AQ-MsgTopicPay1-异步-消息",
	})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "序列化 AQ TopicPay1 消息失败", err)
	}
	if err = d.ctx.MsgPub.SyncCtx(ctx, global.EventTopicPay1, string(demoMsg4)); err != nil {
		return err
	}

	demoMsg5, err := json.Marshal(&global.PayRecharge{
		TradeNo:    "O105",
		PayType:    global.PayTypeWxpay,
		Status:     global.PayStatusFail,
		UserId:     0,
		Cash:       0,
		CreateTime: utils.TimeStr(time.Now()),
		Remark:     "AQ-每三秒一次-消息",
	})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "序列化定时消息失败", err)
	}
	if err = d.ctx.MsgPub.Cron(global.EventTopicPay, string(demoMsg5), "@every 3s"); err != nil {
		return err
	}

	return nil
}

// 手动添加消息
func (d *demoSvc) AddMsg(ctx context.Context) error {
	demoMsg0, err := json.Marshal(&global.PayRecharge{
		TradeNo:    utils.GenStr(6, 8),
		PayType:    global.PayTypeAlipay,
		Status:     global.PayStatusIng,
		UserId:     123,
		Cash:       utils.Yuan2Cent(9.99),
		CreateTime: utils.TimeStr(time.Now()),
		Remark:     "Kafka-无type-手动添加消息",
	})
	if err != nil {
		return xerror.NewError(xerror.BusinessError, "序列化 Kafka 消息失败", err)
	}
	if err = d.ctx.MsgPub.KafkaCtx(ctx, global.EventTopicPay, demoMsg0); err != nil {
		return err
	}
	return nil
}

// 发送删除消息
func (d *demoSvc) DelMsg(ctx context.Context) error {
	_ = d.ctx.MsgPub.DelCron(global.EventTopicPay)
	return nil
}

func (d *demoSvc) ExportStream(ctx context.Context, handler func(*Demo) error) error {
	return d.ctx.DemoDb.FindStream(ctx, func(dbItem *db.Demo) error {
		var item Demo
		if err := copier.Copy(&item, dbItem); err != nil {
			return err
		}
		return handler(&item)
	})
}

func (d *demoSvc) SSE(ctx context.Context, handler func(*SSEMessage) error) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			// 数据处理逻辑（数据格式化和构造完全写在 Service 层）
			msg := &SSEMessage{
				Time: t.Format("2006-01-02 15:04:05"),
				Msg:  "hello sse",
			}
			if err := handler(msg); err != nil {
				return err
			}
		}
	}
}

// wsOut websocket 写出消息
type wsOut struct {
	msgType int
	payload []byte
}

func (d *demoSvc) WS(ctx context.Context, conn *websocket.Conn) error {
	// gorilla/websocket 不支持并发写,所有写操作都经 writeCh 串行化由单个写协程执行
	writeCh := make(chan wsOut, 16)
	done := make(chan struct{})

	// 写协程:唯一调用 conn.WriteMessage 的地方
	writeCtx, writeCancel := context.WithCancel(ctx)
	defer writeCancel()
	go func() {
		for {
			select {
			case <-writeCtx.Done():
				return
			case out, ok := <-writeCh:
				if !ok {
					return
				}
				if err := conn.WriteMessage(out.msgType, out.payload); err != nil {
					xlog.Error(ctx, "websocket write failed", err)
					writeCancel()
					return
				}
			}
		}
	}()

	// 接收消息并 Echo
	go func() {
		defer close(done)
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				xlog.Info(ctx, fmt.Sprintf("websocket read failed or closed: %v", err))
				return
			}
			xlog.Info(ctx, fmt.Sprintf("websocket received message: %s", string(message)))
			// 收到消息后，加上前缀 Echo 返回客户端
			echoMsg := fmt.Sprintf("echo: %s", string(message))
			select {
			case writeCh <- wsOut{msgType: messageType, payload: []byte(echoMsg)}:
			case <-writeCtx.Done():
				return
			}
		}
	}()

	// 模拟服务器主动推送:连接后立即下发一条随机字符串消息
	go func() {
		msg := fmt.Sprintf("server push: %s", utils.GenStr(utils.AlphaNoLowerZeroDigital, 16))
		select {
		case writeCh <- wsOut{msgType: websocket.TextMessage, payload: []byte(msg)}:
		case <-writeCtx.Done():
		}
	}()

	// 定时发送消息给客户端
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			msg := fmt.Sprintf("server time: %s", t.Format("2006-01-02 15:04:05"))
			select {
			case writeCh <- wsOut{msgType: websocket.TextMessage, payload: []byte(msg)}:
			case <-writeCtx.Done():
				return nil
			}
		}
	}
}

func (d *demoSvc) ExportExcel(ctx context.Context, writer io.Writer) error {
	// 创建 Excel 文件并使用 StreamWriter 流式写入
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "Demo数据"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)
	// 删除默认的 Sheet1
	_ = f.DeleteSheet("Sheet1")

	// 创建流式写入器
	sw, err := f.NewStreamWriter(sheetName)
	if err != nil {
		return err
	}

	// 写入表头
	headers := []any{"ID", "名称", "测试1", "测试4"}
	if err := sw.SetRow("A1", headers); err != nil {
		return err
	}

	// 5. 开始流式导出数据
	rowNum := 2
	err = d.ExportStream(ctx, func(item *Demo) error {
		// 写入数据行
		rowData := []any{item.Id, item.Name, item.Test1, item.Test4}
		if err := sw.SetRow(fmt.Sprintf("A%d", rowNum), rowData); err != nil {
			return err
		}
		rowNum++
		return nil
	})

	if err != nil {
		return err
	}

	// 结束流式写入
	if err := sw.Flush(); err != nil {
		return err
	}

	// 设置表头样式
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#E0E0E0"},
		},
	})
	_ = f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), style)

	// 设置列宽
	_ = f.SetColWidth(sheetName, "A", "A", 10)
	_ = f.SetColWidth(sheetName, "B", "B", 30)
	_ = f.SetColWidth(sheetName, "C", "C", 15)
	_ = f.SetColWidth(sheetName, "D", "D", 15)

	// 将完整的 Excel 文件写入传入的 writer
	if err := f.Write(writer); err != nil {
		return err
	}
	return nil
}
