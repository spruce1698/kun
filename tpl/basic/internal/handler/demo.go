package handler

import (
	"basic/internal/service/svc"
	"basic/pkg/xerror"
	"basic/pkg/xhttp"

	"github.com/jinzhu/copier"

	"github.com/gin-gonic/gin"
)

type (
	DemoHandler struct {
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
func (d *DemoHandler) Detail(ctx *gin.Context) {
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
func (d *DemoHandler) List(ctx *gin.Context) {
	req := &struct {
		xhttp.ReqPage
		Id   *int64 `form:"id"   json:"id"`
		Name string `form:"name"  json:"name"` // 名称
	}{}
	if err := ctx.ShouldBind(req); err != nil {
		xhttp.BusCode(ctx, xerror.ParamError, err)
		return
	}

	args := &svc.DemoListArgs{}
	_ = copier.Copy(args, req)

	// id= (空字符串) 时 gin 会把 *int64 绑定成指向 0 的非空指针,
	// 导致 WHERE id = 0 查不到数据。这里把 0 规整为 nil,表示不按 id 过滤。
	if args.Id != nil && *args.Id == 0 {
		args.Id = nil
	}

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
func (d *DemoHandler) Create(ctx *gin.Context) {
	req := &struct {
		Name  string  `form:"name"  json:"name"  binding:"required"` // 名称
		Test1 float64 `form:"test1"  json:"test1"`                   // 测试1
		Test4 int32   `form:"test4"  json:"test4"`                   // 测试4
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
	xhttp.Success(ctx, "Create Demo 成功")
}

// @Summary Update
// @Description 更新一个demo,有事务
// @Tags api
// @Accept json
// @Produce json
// @Success 200 {string} string "{"code":0,"message":"demo-Demo成功","data":{"id":100,"name":"我是demo名称"}}"
// @Failure 500 {string} string "{"code":50000,"message":"demo查询失败","data":""}"
// @Router /api/demo/update [post]
func (d *DemoHandler) Update(ctx *gin.Context) {
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
	xhttp.Success(ctx, "Update Demo 成功")
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
func (d *DemoHandler) Delete(ctx *gin.Context) {
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
	xhttp.Success(ctx, "Delete Demo 成功")
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
func (d *DemoHandler) SoftDelete(ctx *gin.Context) {
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
	xhttp.Success(ctx, "SoftDelete Demo 成功")
}
