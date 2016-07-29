# log-websockt
基于 github.com/gorilla/websocket 实现的websocket    

#### 安装
`go get github.com/bupt1987/log-websockt`

~~~~
开启websocket 服务,端口: 9090
本地监听 /tmp/log-stock.socket ,把输入的文本信息push到连接到websocket的客户端

往本地socket写入数据时需要注意, 数据必须以 "\n" 为一条log的结束

~~~~
