# websocket
```go
//全局实例化
ISCWsManager := ws.NewManager()


//handler中调用Serve
ISCWsManager.Serve(w, r, func(conn *websocket.Conn, manager *ws.Manager) (ws.IClient, error) {
    c := ws.NewGoroutineClient(conn, manager, svc.ISCHandler(svcCtx))
    return c, nil
})
```