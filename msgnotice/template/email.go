package template

const MailAccount = "zhiwei@dc-science.com"

const (
	SAFE_STOCK_NOTICE_TPL = `<div>【{{.WarehouseName}}】中【{{.AssetTypeName}}】类型的【{{.AssetModelName}}】已不足设置安全库存【{{.SateStockCnt}}】，请及时补充库存。</div>`
)
