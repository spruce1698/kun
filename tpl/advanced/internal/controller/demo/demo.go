package demo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"advanced/internal/global"
	"advanced/internal/service/svc"
	"advanced/pkg/xerror"
	"advanced/pkg/xhttp"
	"advanced/pkg/xlog"

	"github.com/jinzhu/copier"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/websocket"
)

type (
	DemoCtrl struct {
		DemoSvc svc.DemoSvc
	}
)

// @Summary Detail
// @Description demo详情
// @Tags api
// @Accept json
// @Produce json
// @Param id query int true "demoID"
// @Success 200 {string} string "{"code":0,"message":"demo-Demo成功","data":{"id":100,"name":"我是demo名称"}}"
// @Failure 500 {string} string "{"code":50000,"message":"demo查询失败","data":""}"
// @Router /api/demo/:id [get]
func (d *DemoCtrl) Detail(ctx *gin.Context) {
	// 获取参数两种方式
	// 方式1:
	// idStr := ctx.Param("id")
	// if idStr == "" {
	// 	xhttp.BusCode(ctx, xerror.ParamError, errors.New("id不能为空"))
	// 	return
	// }
	// id, _ := strconv.ParseInt(idStr, 10, 64)

	//  方式2(推荐可以方便验证参数):
	type detailId struct {
		Id int64 `uri:"id" binding:"required,min=1,max=100"` // 或者 binding:"required,gte=1,lte=130"
	}
	var urlId detailId
	if err := ctx.ShouldBindUri(&urlId); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}
	type minMaxStruct struct {
		Min string `form:"min" binding:"required,minNow"`
		Max string `form:"max" binding:"required,maxNow"` // 自定义校验
	}
	var minMax minMaxStruct
	if err := ctx.ShouldBind(&minMax); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}
	id := urlId.Id

	info, err := d.DemoSvc.Detail(ctx.Request.Context(), id)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "Get Demo Detail 成功", info)
}

// @Summary List
// @Description 根据条件查询demo列表
// @Tags api
// @Accept json
// @Produce json
// @Success 200 {string} string "{"code":0,"message":"demo-Demo成功","data":{"id":100,"name":"我是demo名称"}}"
// @Failure 500 {string} string "{"code":50000,"message":"demo查询失败","data":""}"
// @Router /api/demo/list [get]
func (d *DemoCtrl) List(ctx *gin.Context) {
	req := &struct {
		xhttp.PageArg
		Id   *int64 `form:"id"   json:"id"`
		Name string `form:"name"  json:"name"` // 名称
	}{}
	if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}

	args := &svc.DemoListArgs{}
	_ = copier.Copy(args, req)

	result, total, err := d.DemoSvc.FindList(ctx.Request.Context(), args)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.List(ctx, "Get Demo List 成功", req.Page, req.PageSize, total, result)
}

// @Summary Create
// @Description 写一个demo
// @Tags api
// @Accept json
// @Produce json
// @Param id query int true "demoID"
// @Success 200 {string} string "{"code":0,"message":"demo-Demo成功","data":{"id":100,"name":"我是demo名称"}}"
// @Failure 500 {string} string "{"code":50000,"message":"demo查询失败","data":""}"
// @Router /api/demo/create [post]
func (d *DemoCtrl) Create(ctx *gin.Context) {
	req := &struct {
		Name     string  `form:"name"  json:"name"  binding:"required"` // 名称
		Test1    float64 `form:"test1"  json:"test1"`                   // 测试1
		Test4    int32   `form:"test4"  json:"test4"`                   // 测试4
		RoleId   int64   `form:"roleId" json:"roleId"`                  // 角色id
		Password string  `form:"password" json:"password"`              // 密码(明文,空则默认123456)
	}{}
	if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}
	args := &svc.Demo{}
	_ = copier.Copy(args, req)
	err := d.DemoSvc.Create(ctx.Request.Context(), args)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "Create Demo 成功", nil)
}

// @Summary Update
// @Description 更新一个demo,有事务
// @Tags api
// @Accept json
// @Produce json
// @Success 200 {string} string "{"code":0,"message":"demo-Demo成功","data":{"id":100,"name":"我是demo名称"}}"
// @Failure 500 {string} string "{"code":50000,"message":"demo查询失败","data":""}"
// @Router /api/demo/update [post]
func (d *DemoCtrl) Update(ctx *gin.Context) {
	req := &struct {
		Id    int64   `json:"id"   binding:"required"` //
		Name  string  `json:"name"`                    // 名称
		Test1 float64 `json:"test1"`                   // 测试1
		Test4 int32   `json:"test4"`                   // 测试4
	}{}
	if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}
	args := &svc.Demo{}
	_ = copier.Copy(args, req)
	err := d.DemoSvc.Update(ctx.Request.Context(), args)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "Update Demo 成功", nil)
}

// @Summary Delete
// @Description 删除一个demo
// @Tags api
// @Accept json
// @Produce json
// @Param id query int true "demoID"
// @Success 200 {string} string "{"code":0,"message":"demo-Demo成功","data":{"id":100,"name":"我是demo名称"}}"
// @Failure 500 {string} string "{"code":50000,"message":"demo查询失败","data":""}"
// @Router /demo/delete [post]
func (d *DemoCtrl) Delete(ctx *gin.Context) {
	req := &struct {
		Id int64 `form:"id" json:"id" binding:"required,gte=1,lte=130"`
	}{}
	if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}

	err := d.DemoSvc.Delete(ctx.Request.Context(), req.Id)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "Delete Demo 成功", nil)
}

// @Summary SoftDelete
// @Description 逻辑删除一个demo
// @Tags api
// @Accept json
// @Produce json
// @Param id query int true "demoID"
// @Success 200 {string} string "{"code":0,"message":"demo-Demo成功","data":{"id":100,"name":"我是demo名称"}}"
// @Failure 500 {string} string "{"code":50000,"message":"demo查询失败","data":""}"
// @Router /demo/softdelete [post]
func (d *DemoCtrl) SoftDelete(ctx *gin.Context) {
	req := &struct {
		Id int64 `form:"id" json:"id" binding:"required,gte=1,lte=130"`
	}{}
	if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}

	err := d.DemoSvc.SoftDelete(ctx.Request.Context(), req.Id)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "SoftDelete Demo 成功", nil)
}

// @Summary Login
// @Description demo登录(账号:name,密码:password),返回双token
// @Tags api
// @Accept json
// @Produce json
// @Param name formData string true "账号(demo名称)"
// @Param password formData string true "密码"
// @Success 200 {string} string "{"code":0,"message":"Login 成功","data":{"accessToken":"","refreshToken":""}}"
// @Router /api/auth/login [post]
func (d *DemoCtrl) Login(ctx *gin.Context) {
	req := &struct {
		Name     string `form:"name" json:"name" binding:"required"`         // 账号(demo名称)
		Password string `form:"password" json:"password" binding:"required"` // 密码(123456)
	}{}
	if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}
	tokenInfo, err := d.DemoSvc.Login(ctx.Request.Context(), req.Name, req.Password)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "Login 成功", tokenInfo)
}

// @Summary Refresh
// @Description 刷新双token
// @Tags api
// @Accept json
// @Produce json
// @Param accessToken formData string true "访问token"
// @Param refreshToken formData string true "刷新token"
// @Router /api/auth/refresh [post]
func (d *DemoCtrl) Refresh(ctx *gin.Context) {
	req := &struct {
		AccessToken  string `form:"accessToken" json:"accessToken" binding:"required"`
		RefreshToken string `form:"refreshToken" json:"refreshToken" binding:"required"`
	}{}
	if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}
	tokenInfo, err := d.DemoSvc.RefreshToken(ctx.Request.Context(), req.AccessToken, req.RefreshToken)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "Refresh 成功", tokenInfo)
}

// @Summary Info
// @Description 当前登录信息(需授权),从token中解析userid和roleid
// @Tags api
// @Produce json
// @Router /api/auth/info [get]
func (d *DemoCtrl) Info(ctx *gin.Context) {
	userId, userIdExists := ctx.Get(global.CtxAuthUserId)
	roleId, roleIdExists := ctx.Get(global.CtxAuthRoleId)
	if !userIdExists || !roleIdExists {
		xhttp.BusCode(ctx, xerror.Unauthorized, fmt.Errorf("用户未登录或认证信息缺失"))
		return
	}
	xhttp.Data(ctx, "Get Info 成功", gin.H{
		"userId": userId,
		"roleId": roleId,
	})
}

// @Summary Logout
// @Description 登出,废弃当前token
// @Tags api
// @Accept json
// @Produce json
// @Router /api/auth/logout [post]
func (d *DemoCtrl) Logout(ctx *gin.Context) {
	tokenStr := ctx.GetString(global.CtxAuthToken)
	if err := d.DemoSvc.Logout(ctx.Request.Context(), tokenStr); err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "Logout 成功", nil)
}

func (d *DemoCtrl) SendMsg(ctx *gin.Context) {
	err := d.DemoSvc.SendMsg(ctx.Request.Context())
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "SenMsg 成功", nil)
}

func (d *DemoCtrl) AddMsg(ctx *gin.Context) {
	err := d.DemoSvc.AddMsg(ctx.Request.Context())
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "AddMsg 成功", nil)
}

func (d *DemoCtrl) DelMsg(ctx *gin.Context) {
	err := d.DemoSvc.DelMsg(ctx.Request.Context())
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "DelMsg 成功", nil)
}

func (d *DemoCtrl) GetCsrf(ctx *gin.Context) {
	csrfToken := csrf.Token(ctx.Request)
	xhttp.Data(ctx, "GetCsrf 成功", csrfToken)
}

func (d *DemoCtrl) SubmitCsrf(ctx *gin.Context) {
	xhttp.Data(ctx, "SubmitCsrf 成功", "")
}

// @Summary Export
// @Description http+mysql 全链路流式导出 demo
// @Tags api
// @Accept json
// @Produce json
// @Router /api/demo/export [get]
func (d *DemoCtrl) Export(ctx *gin.Context) {
	// 1. 清除 WriteDeadline，支持长连接流式传输
	rc := http.NewResponseController(ctx.Writer)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		// 忽略或记录日志
	}

	// 2. 设置流式响应头
	ctx.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	ctx.Header("X-Accel-Buffering", "no") // 禁止 nginx buffer
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Status(http.StatusOK)

	// 3. 获取 Flusher
	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		// 若不支持 flusher，则回退或直接报错
		return
	}
	flusher.Flush()

	// 4. 开始流式导出
	var count int64
	err := d.DemoSvc.ExportStream(ctx.Request.Context(), func(item *svc.Demo) error {
		line, err := json.Marshal(item)
		if err != nil {
			return err
		}
		_, err = ctx.Writer.Write(append(line, '\n'))
		if err != nil {
			return err
		}
		count++
		flusher.Flush()
		return nil
	})

	// 5. 哨兵行协议写入
	var sentinel []byte
	if err != nil {
		sentinel, _ = json.Marshal(map[string]any{
			"_end":     "err",
			"exported": count,
			"err":      err.Error(),
		})
	} else {
		sentinel, _ = json.Marshal(map[string]any{
			"_end":     "ok",
			"exported": count,
		})
	}
	_, _ = ctx.Writer.Write(append(sentinel, '\n'))
	flusher.Flush()
}

// @Summary Excel
// @Description http+mysql 全链路流式导出 Excel，浏览器直接下载
// @Tags api
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Router /api/demo/excel [get]
func (d *DemoCtrl) Excel(ctx *gin.Context) {
	// 1. 清除 WriteDeadline，支持长连接流式传输
	rc := http.NewResponseController(ctx.Writer)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		// 忽略或记录日志
	}

	// 2. 设置 Excel 下载响应头
	fileName := fmt.Sprintf("demo_export_%s.xlsx", time.Now().Format("20060102150405"))
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", fileName))
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("X-Accel-Buffering", "no") // 禁止 nginx buffer
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Status(http.StatusOK)

	// 3. 获取 Flusher
	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		// 若不支持 flusher，则回退或直接报错
		xhttp.BusCode(ctx, xerror.BusinessError, fmt.Errorf("streaming not supported"))
		return
	}
	flusher.Flush()

	// 4. 调用 Svc 层进行 Excel 流式写入
	err := d.DemoSvc.ExportExcel(ctx.Request.Context(), ctx.Writer)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	flusher.Flush()
}

// @Summary SSE
// @Description Server-Sent Events demo
// @Tags api
// @Produce text/event-stream
// @Router /api/demo/sse [get]
func (d *DemoCtrl) SSE(ctx *gin.Context) {
	// 1. 清除 WriteDeadline,支持长连接流式传输(否则会被 http.Server.WriteTimeout 切断)
	rc := http.NewResponseController(ctx.Writer)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		// 忽略或记录日志
	}

	// 2. 设置响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no") // 防止 nginx 缓存
	ctx.Status(http.StatusOK)

	flusher, ok := ctx.Writer.(http.Flusher)
	if ok {
		flusher.Flush()
	}

	// 3. 调用 Svc 层进行消息发送
	_ = d.DemoSvc.SSE(ctx.Request.Context(), func(msg *svc.SSEMessage) error {
		ctx.SSEvent("message", msg)
		// gin 的 SSEvent 不保证立即刷出,这里显式 Flush 让客户端实时收到
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	})
}

// @Summary WSTicket
// @Description 申请 WebSocket 一次性升级票据(需授权),用 ticket 连接 WS,避免长 token 进 URL
// @Tags api
// @Produce json
// @Router /api/auth/ws-ticket [post]
func (d *DemoCtrl) WSTicket(ctx *gin.Context) {
	uid, _ := ctx.Get(global.CtxAuthUserId)
	rid, _ := ctx.Get(global.CtxAuthRoleId)
	userId, _ := uid.(int64)
	roleId, _ := rid.(int64)
	ticket, err := d.DemoSvc.WSTicket(ctx.Request.Context(), userId, roleId)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xhttp.Data(ctx, "WSTicket 成功", gin.H{"ticket": ticket})
}

// @Summary WebSocket Demo
// @Description WebSocket 示例，支持 Echo 及定时心跳数据推送(需 ticket 鉴权)
// @Tags api
// @Param ticket query string true "一次性升级票据(由 /api/auth/ws-ticket 获取)"
// @Router /api/demo/ws [get]

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 拒绝无 Origin 请求(浏览器同源策略保证浏览器必定携带 Origin),
		// 非浏览器客户端应通过 ticket 鉴权接入,不走 CheckOrigin
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}
		// 带了 Origin 的(浏览器)请求,校验同源,防 CSRF。
		return origin == fmt.Sprintf("%s://%s", func() string {
			if r.TLS != nil {
				return "https"
			}
			return "http"
		}(), r.Host)
	},
}

func (d *DemoCtrl) WS(ctx *gin.Context) {
	// WS 升级无法携带自定义 Header,用一次性 ticket 鉴权,长 token 不进 URL
	ticket := ctx.Query("ticket")
	userId, roleId, err := d.DemoSvc.ConsumeWSTicket(ctx.Request.Context(), ticket)
	if err != nil {
		xhttp.BusFail(ctx, err)
		return
	}
	xlog.Info(ctx.Request.Context(), fmt.Sprintf("websocket connected, userId=%d roleId=%d", userId, roleId))

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		xlog.Error(ctx.Request.Context(), "websocket upgrade failed", err)
		return
	}
	defer conn.Close()

	if err := d.DemoSvc.WS(ctx.Request.Context(), conn); err != nil {
		xlog.Error(ctx.Request.Context(), "websocket session failed", err)
	}
}
