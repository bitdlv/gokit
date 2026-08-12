# 简单存储
```go
s := storage.NewStorage(deivers.NewLocal("."), storage.WithUrlPrefix("http://www.baidu.com/attachments"))
s.Put(image, "/cache/file.jpg")
```