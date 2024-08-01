一个内网穿透工具，可直接暴露内网端口到公网，或者暴露socks5代理以访问内网

## 构建
```bash
go generate ./... //生成证书
go build -o tunnel ./cmd/tunnel
```
## 使用
服务端
```bash
./tunnel server -l :10086 -t 1111 //监听在10086端口，token为1111
```
客户端
```bash
//将服务器的12345端口映射到内网127.0.0.1:3306，并暴露了socks5代理在服务端10087端口
./tunnel client -s 127.0.0.1:10086  -t 1111 -p 10087 -r 12345/example.com:443
```