# errx

errx包提供一种错误类型。
这个类型能记录简单的错误栈，还提供将自己转换成 *status.Status 的方法。
在grpc方法中返回本包的错误类型，grpc将自动把err转换为 *status.Status。
在传输过程中不会丢失错误信息(包括错误栈)。

本包提供:
  * 包裹方法 Wrapxxx
  * 实例化方法 Newxxx，特别支持从 *status.Status 读取信息并实例化。

errx.Error 类型还记录了实例化时的文件名和行号。
如果用errx.Error包裹errx.Error，还能获得简单的错误栈。

打印错误消息时使用#v可打印错误栈，使用+v可打印错误消息链，使用v或者调用Error()方法，只打印自身的错误消息。