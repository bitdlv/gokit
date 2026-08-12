# 八、消息 / 通知

## mail — SMTP 邮件（gomail）

**类型**：`Mail`、`MailConf`、`MailPool`

```go
m := mail.NewMail(cfg)
err := m.From("a@x").To("b@y").
    AddToCC("c@y").
    SetSubject("hi").
    Native("<h1>hello</h1>").
    Attach("/tmp/report.pdf").
    Send()

// 连接池
p := mail.NewMailPool(cfg, 5)
p.SendMail(mail)
p.SendTemplate(tpl, data)
```

| API | 说明 |
|---|---|
| `NewMail(cfg)` | 单邮件 |
| `NewMailPool(cfg, size)` | 连接池 |
| `From(addr) / To(addr) / AddToCC(addr)` | 收发件 |
| `SetSubject(s)` | 标题 |
| `Native(html)` | HTML 正文 |
| `Attach(path)` | 附件 |
| `Send()` | 发送 |
| `SendMail / SendTemplate / SendMailByTemplate` | 池发送 |

## msgnotice — 业务告警模板（云片短信 + 邮件）

**类型**：
- `Mailer` / `YunPianSMSClient`
- 短信体：`AlarmDoSmsBody` / `AlarmUndoSmsBody` / `AlarmGenerateSmsBody` / `StockAlarmNoticeSmsBody`

**子包**：`msgnotice/template` — 内置模板文案。

```go
smsCli := msgnotice.NewYunPianSMSClient(apikey)
smsCli.SetTpl("alarm_do").SetValue(map[string]string{"name":"X"}).SetTo("13800000000").SendSMS()

m := msgnotice.NewMailer(cfg)
m.SetFrom("ops").SetTo("dev@x").SetSubject("告警").SetAlarmDoSmsBody(body).Send()
```

| API | 说明 |
|---|---|
| `NewMailer(cfg)` / `NewYunPianSMSClient(key)` | 构造 |
| `Send() / SendSMS()` | 发送 |
| `SetFrom / SetCustomFrom / SetTo / SetSubject` | 收发件 |
| `SetTpl / SetValue` | 模板 + 变量 |
| `SetAlarmDoSmsBody / SetAlarmUndoContext / SetAlarmGenerateSmsBody / SetStockAlarmNoticeSmsBody` | 内置告警体 |

## pushgateway — 统一推送网关代理

**类型**：`Sender`

```go
s := pushgateway.NewPushGatewaySender(url, token)
s.SendEmail(to, subject, body)
s.SendSMS(mobile, tpl, args)
s.SendInstantNotice(userId, title, content)
```

## senders — Slack / 日志式发送器

**类型**：`Sender`、`LogSender`、`SlackSender`、`SlackMessage`、`Notification`

```go
sender := senders.SlackSender{Webhook: url}
sender.Send(senders.SlackMessage{Text: "hi"})
```

| API | 说明 |
|---|---|
| `BuildMessage(...)` | 构造消息 |
| `Send(msg)` | 发送单条 |
| `SingleSend(...)` | 单发助手 |
| `IsSupport(kind)` | 支持度 |

## 分层建议

| 场景 | 用哪个 |
|---|---|
| SMTP 直发 | `mail` |
| 业务告警模板 | `msgnotice` |
| 统一网关代理 | `pushgateway` |
| Slack / 日志 | `senders` |

## 测试

```bash
# 全部用 mock http.RoundTripper
go test -v ./mail/... ./msgnotice/... ./pushgateway/... ./senders/...
```
