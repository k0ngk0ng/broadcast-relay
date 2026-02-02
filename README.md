# Broadcast Relay

一个跨平台的 UDP 广播包转发中继器，支持将本地接收到的广播包转发到指定的 IP:Port。

## 功能特性

- ✅ 支持 macOS 和 Windows
- ✅ 单一可执行文件，即开即用
- ✅ 支持多目标转发
- ✅ 详细的统计信息
- ✅ 低资源占用

## 下载

从 [Releases](https://github.com/k0ngk0ng/broadcast-relay/releases) 页面下载对应平台的可执行文件：

- `broadcast-relay-windows-amd64.exe` - Windows 64位
- `broadcast-relay-darwin-amd64` - macOS Intel
- `broadcast-relay-darwin-arm64` - macOS Apple Silicon (M1/M2/M3)

## 使用方法

### 基本用法

```bash
# 监听 9999 端口，转发到 192.168.1.100:9999
./broadcast-relay -port 9999 -targets 192.168.1.100:9999

# Windows
broadcast-relay.exe -port 9999 -targets 192.168.1.100:9999
```

### 多目标转发

```bash
# 转发到多个目标
./broadcast-relay -port 9999 -targets 192.168.1.100:9999,10.0.0.50:8888
```

### 详细模式

```bash
# 启用详细日志
./broadcast-relay -port 9999 -targets 192.168.1.100:9999 -verbose
```

### 所有参数

```
Usage: broadcast-relay [options]

Options:
  -port int
        UDP port to listen for broadcast packets (default 9999)
  -listen string
        Address to listen on (use 0.0.0.0 for all interfaces) (default "0.0.0.0")
  -targets string
        Comma-separated list of target addresses (ip:port), e.g., 192.168.1.100:9999,10.0.0.50:8888
  -buffer int
        UDP buffer size in bytes (default 65535)
  -verbose
        Enable verbose logging
  -version
        Show version information
```

## 使用场景

### 场景 1: 游戏局域网联机

某些游戏使用 UDP 广播进行局域网发现，但广播包无法跨网段。使用此工具可以将广播包转发到其他网段的机器。

```bash
# 在网关机器上运行
./broadcast-relay -port 27015 -targets 192.168.2.255:27015
```

### 场景 2: IoT 设备发现

智能家居设备通常使用广播进行发现，使用此工具可以跨 VLAN 发现设备。

```bash
./broadcast-relay -port 1900 -targets 10.0.1.255:1900,10.0.2.255:1900
```

### 场景 3: 开发调试

在开发网络应用时，将广播包转发到测试服务器进行调试。

```bash
./broadcast-relay -port 8888 -targets 192.168.1.100:8888 -verbose
```

## 编译

### 本地编译

```bash
# 编译当前平台
go build -o broadcast-relay .

# 交叉编译 Windows
GOOS=windows GOARCH=amd64 go build -o broadcast-relay.exe .

# 交叉编译 macOS Intel
GOOS=darwin GOARCH=amd64 go build -o broadcast-relay-darwin-amd64 .

# 交叉编译 macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o broadcast-relay-darwin-arm64 .
```

### 使用 Makefile

```bash
make build      # 编译当前平台
make all        # 编译所有平台
make clean      # 清理编译产物
```

## Windows 防火墙设置

在 Windows 上首次运行时，可能需要允许防火墙访问：

1. 运行程序时会弹出防火墙提示，点击"允许访问"
2. 或者手动添加防火墙规则：
   ```powershell
   netsh advfirewall firewall add rule name="Broadcast Relay" dir=in action=allow program="C:\path\to\broadcast-relay.exe" enable=yes
   ```

## macOS 权限设置

在 macOS 上，下载的二进制文件可能需要解除隔离：

```bash
# 解除隔离属性
xattr -d com.apple.quarantine broadcast-relay-darwin-*

# 添加执行权限
chmod +x broadcast-relay-darwin-*
```

## License

MIT License
