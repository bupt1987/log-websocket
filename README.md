# log-websocket
监听本地socket ,把输入的文本信息push到连接到websocket的客户端
websocket 基于 github.com/gorilla/websocket  

## 安装
`go get github.com/bupt1987/log-websocket`

## 使用须知
~~~~
websocket 服务,端口: 9090, 访问需要带上参数listens, 例如: ws://127.0.0.1/?listens=test1,test2
listens 后面接的是监听的消息类型,可同时监听多种消息类型, 如果是 "*" 的话表示监听所有类型的消息

往本地 /tmp/log-stock.socket 写入数据时需要注意
本地写入数据格式: 消息类型 + "," + 消息内容 + "\n"

例子都在 tests 目录下

~~~~
