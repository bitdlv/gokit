package msgnotice

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	YUNPIAN_TPL_ID_ALARMUNDO      = 5840254
	YUNPIAN_TPL_ID_ALARMDO        = 5840252
	YUNPIAN_TPL_ID_ALARM_GENERATE = 5840246

	YUNPIAN_TPL_ID_STOCK_NOTICE = 5840246
)

// AlarmUndoSmsBody
// tpl_id 5840254
// 您有1条告警通知！
// #devicename#  #devocelocation# 于 #alarmtime# 产生 #alarmlevel# 已持续 #alarmduration# #alarmundostatus#，请知悉。
type AlarmUndoSmsBody struct {
	DeviceName      string //设备名称
	DevoceLocation  string //设备位置
	AlarmTime       string //告警时间
	AlarmLevel      string //告警等级
	AlarmDuration   string //周期
	AlarmUndoStatus string //告警未处理状态
}

// AlarmGenerateSmsBody 您有1条告警通知！
// #DeviceName# #DevoceLocation# 于 #AlarmTime# 产生 #AlarmLevel# 请熟悉。
type AlarmGenerateSmsBody struct {
	DeviceName     string
	DevoceLocation string
	AlarmTime      string
	AlarmLevel     string
}

// AlarmDoSmsBody 您有1条告警通知！
// #DeviceName# #DevoceLocation# 于 #AlarmTime# 产生 #AlarmLevel# 于 #AlarmRecoverTime# 已被 #AlarmProcessor# #AlarmDoStatus#，请知悉。//处理恢复
type AlarmDoSmsBody struct {
	DeviceName       string
	DevoceLocation   string
	AlarmTime        string
	AlarmLevel       string
	AlarmRecoverTime string
	AlarmProcessor   string
	AlarmDoStatus    string
}

// StockAlarmNoticeSmsBody 您有1条告警通知！
// #WarehouseName#中#AssetTypeName#类型的#AssetModelName#已不足设置安全库存#SateStockCnt#，请及时补充库存。//安全库存告警通知
type StockAlarmNoticeSmsBody struct {
	WarehouseName  string
	AssetModelName string
	AssetTypeName  string
	SateStockCnt   string
}

type YunPianSMSClient struct {
	c         *http.Client
	tplId     string
	apiKey    string
	urlTplSms string
	context   string
}

func NewYunPianSMSClient(apikey, tplSmsUrl string) *YunPianSMSClient {
	return &YunPianSMSClient{
		apiKey:    apikey,
		urlTplSms: tplSmsUrl,
		c: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,               // 最大空闲连接数
				IdleConnTimeout:     30 * time.Second, // 空闲连接超时时间
				MaxIdleConnsPerHost: 10,
			},
		},
	}
}

func (l *YunPianSMSClient) SetAlarmUndoContext(tplId int, params AlarmUndoSmsBody) *YunPianSMSClient {
	tplValue := url.Values{
		"#devicename#":      {params.DeviceName},
		"#devocelocation#":  {params.DevoceLocation},
		"#alarmtime#":       {params.AlarmTime},
		"#alarmlevel#":      {params.AlarmLevel},
		"#alarmduration#":   {params.AlarmDuration},
		"#alarmundostatus#": {params.AlarmUndoStatus},
	}.Encode()
	l.context = tplValue
	l.tplId = fmt.Sprintf("%d", tplId)
	return l
}

// SetAlarmGenerateSmsBody
// #DeviceName# #DevoceLocation# 于 #AlarmTime# 产生 #AlarmLevel# 请熟悉。
func (l *YunPianSMSClient) SetAlarmGenerateSmsBody(tplId int, params AlarmGenerateSmsBody) *YunPianSMSClient {
	tplValue := url.Values{
		"#DeviceName#":     {params.DeviceName},
		"#DevoceLocation#": {params.DevoceLocation},
		"#AlarmTime#":      {params.AlarmTime},
		"#AlarmLevel#":     {params.AlarmLevel},
	}.Encode()
	l.context = tplValue
	l.tplId = fmt.Sprintf("%d", tplId)
	return l
}

// SetAlarmDoSmsBody
// #DeviceName# #DevoceLocation# 于 #AlarmTime# 产生 #AlarmLevel# 于 #AlarmRecoverTime# 已被 #AlarmProcessor# #AlarmDoStatus#，请知悉。//处理恢复
func (l *YunPianSMSClient) SetAlarmDoSmsBody(tplId int, params AlarmDoSmsBody) *YunPianSMSClient {
	tplValue := url.Values{
		"#DeviceName#":       {params.DeviceName},
		"#DevoceLocation#":   {params.DevoceLocation},
		"#AlarmTime#":        {params.AlarmTime},
		"#AlarmLevel#":       {params.AlarmLevel},
		"#AlarmRecoverTime#": {params.AlarmRecoverTime},
		"#AlarmProcessor#":   {params.AlarmProcessor},
		"#AlarmDoStatus#":    {params.AlarmDoStatus},
	}.Encode()
	l.context = tplValue
	l.tplId = fmt.Sprintf("%d", tplId)
	return l
}

func (l *YunPianSMSClient) SetStockAlarmNoticeSmsBody(tplId int, params StockAlarmNoticeSmsBody) *YunPianSMSClient {
	tplValue := url.Values{
		"#WarehouseName#":  {params.WarehouseName},
		"#AssetModelName#": {params.AssetModelName},
		"#AssetTypeName#":  {params.AssetTypeName},
		"#SateStockCnt#":   {params.SateStockCnt},
	}.Encode()
	l.context = tplValue
	l.tplId = fmt.Sprintf("%d", tplId)
	return l
}

// SendSMS
// 短信发送
func (l *YunPianSMSClient) SendSMS(mobile string) error {
	dataTplSms := url.Values{"apikey": {l.apiKey}, "mobile": {mobile}, "tpl_id": {l.tplId}, "tpl_value": {l.context}}
	err := l.httpsPostForm(l.urlTplSms, dataTplSms)
	if err != nil {
		logx.Errorf("Sending Sms failed: %s", err.Error())
		return err
	}
	return nil
}

func (l *YunPianSMSClient) httpsPostForm(url string, data url.Values) error {
	resp, err := l.c.PostForm(url, data)
	if err != nil {
		logx.Errorf("执行发送短信请求报错：%s", err.Error())
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logx.Errorf("获取发送短信响应内容报错：%s", err.Error())
		return err
	}
	logx.Info("发送短信成功返回: " + string(body))
	return nil
}
