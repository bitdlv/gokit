# xerr包(弃用, 请使用[errx](../errx/readme.md)包)
> 对RPC的返回结果中error定义统一的结构体 （见xerr/error.go）
```go
type CodeError struct {
	errCode uint64
	errMsg  string
}

//返回给前端的错误码
func (e CodeError) GetErrCode() uint64 {
	return e.errCode
}

//返回给前端显示端错误信息
func (e CodeError) GetErrMsg() string {
	return e.errMsg
}

func (e CodeError) Error() string {
	return fmt.Sprintf("ErrCode:%d，ErrMsg:%s", e.errCode, e.errMsg)
}
```

> 对RPC的ErrorCode统一错误码 （见xerr/errCode）
```go
package xerr

//成功返回
const OK uint64 = 200

/**(前3位代表业务,后三位代表具体功能)**/

//全局错误码
const SERVER_COMMON_ERROR uint64 = 100001
const REUQEST_PARAM_ERROR uint64 = 100002
const TOKEN_EXPIRE_ERROR uint64 = 100003
const TOKEN_GENERATE_ERROR uint64 = 100004
const DB_ERROR uint64 = 100005
const DB_UPDATE_AFFECTED_ZERO_ERROR uint64 = 100006

// 维保作业项模块 
const xxxx_ERROR uint64 = 200001 //不同的模块以固定的数字开头便于调别人RPC时发生错误直接定位到代码

// 设备组管理
const xxxx_ERROR uint64 = 300001 //便于调别人RPC时发生错误直接定位到代码

```
> 使用方法： 以维保项的Add logic为例：
```go
// -----------------------维保项-----------------------
func (l *AddItemLogic) AddItem(in *pb.AddItemReq) (*pb.AddItemResp, error) {
    resp := new(pb.AddItemResp)
    itemModel := l.svcCtx.Tnebula.WithContext(l.ctx).DcomFoMaintainItem
    item := model.DcomFoMaintainItem{}
    err := copier.Copy(&item, in)
    if err != nil {
		err = errors.Wrap(xerr.NewErrCodeMsg(xerr.ITEM_COPY_ERROR, "请求序列化输错了"), "copy error!")
        return resp, err

}
....
}

```