package mail

import "testing"

func TestMailPool_SendMail(t *testing.T) {
	templ := `<div>工单号: {{.TicketID}}</div>
		<div>工单类型: {{.MsgType}}</div>`

	ob := map[string]interface{}{"TicketID": "2134", "MsgType": "机架分配"}
	mp := NewMailPool(10, "smtp.exmail.qq.com", 465, "zhiwei@dc-science.com", "j9iptx3uzdTTh6t2")

	mp.SendMailByTemplate("zhiwei@dc-science.com", "miao.chen@dc-science.com", "测试", templ, ob)
	mp.SendMailByTemplate("zhiwei@dc-science.com", "miao.chen@dc-science.com", "测试", templ, ob)
	mp.SendMailByTemplate("zhiwei@dc-science.com", "miao.chen@dc-science.com", "测试", templ, ob)
}
