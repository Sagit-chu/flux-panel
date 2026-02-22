# FLVX WHMCS Provisioning Module

FLVX 面板的 WHMCS 自动开通模块，用于销售流量转发服务。

## 功能特性

- 自动创建 FLVX 用户
- 自动分配隧道
- 流量配额管理
- 服务到期管理
- 客户端服务详情展示
- 支持升级/降级套餐
- 流量使用量自动监控

## 系统要求

### WHMCS 要求
- WHMCS 8.0 或更高版本
- PHP 7.4 或更高版本
- PHP 扩展: cURL, JSON

### FLVX 面板要求
- FLVX 2.1.0 或更高版本
- 已配置 JWT_SECRET
- 已创建至少一个可用隧道

## 安装步骤

### 1. 上传模块文件

将 `flvx` 目录上传到 WHMCS 安装目录的 `/modules/servers/` 下：

```
/path/to/whmcs/
└── modules/
    └── servers/
        └── flvx/
            ├── flvx.php
            ├── lib/
            │   └── FlvxApiClient.php
            └── templates/
                └── overview.tpl
```

### 2. 创建产品自定义字段

在 WHMCS 后台，进入 **Setup > Products/Services > [选择产品] > Custom Fields**，创建以下字段：

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `flvx_user_id` | Text | FLVX 用户ID (系统自动填充) |
| `flvx_username` | Text | FLVX 用户名 |
| `flvx_password` | Text | FLVX 密码 |
| `flvx_tunnel_id` | Text | 分配的隧道ID |
| `flvx_user_tunnel_id` | Text | UserTunnel 关联ID |

### 3. 配置服务器

进入 **Setup > Products/Services > Servers**，添加新服务器：

- **Name:** FLVX Panel
- **Hostname:** 面板地址 (如: panel.example.com)
- **Username:** FLVX 管理员用户名
- **Password:** FLVX 管理员密码
- **Access Hash:** 默认隧道ID (可选)
- **Secure:** 勾选 (如果使用 HTTPS)

### 4. 配置产品

进入 **Setup > Products/Services > [选择产品] > Module Settings**：

- **Module Name:** flvx
- **Traffic Quota (GB):** 流量配额 (默认 100)
- **Max Forwards:** 最大转发数量 (默认 10)
- **Tunnel ID:** 默认隧道ID (留空则使用服务器设置)
- **Speed Limit ID:** 速度限制ID (留空为不限速)
- **Expiry Days:** 服务有效期天数 (默认 30)

## 使用说明

### 开通服务

当客户下单并支付后，WHMCS 会自动调用模块：

1. 在 FLVX 创建用户 (或使用已存在的用户)
2. 将指定隧道分配给用户
3. 设置流量配额和有效期
4. 将登录信息保存到服务自定义字段

### 暂停/恢复服务

- **暂停:** 将用户隧道状态设为禁用
- **恢复:** 将用户隧道状态设为启用

### 终止服务

1. 移除用户的隧道分配
2. 清除服务自定义字段

### 升级/降级

更新用户的流量配额和最大转发数量

## 客户端界面

客户登录后可在产品详情页看到：

- 账户信息 (用户名/密码)
- 隧道信息
- 流量使用情况 (进度条)
- 到期时间
- 快速跳转到 FLVX 面板

## API 端点说明

模块使用以下 FLVX API 端点：

| 端点 | 用途 |
|------|------|
| `/api/v1/user/login` | 获取 JWT Token |
| `/api/v1/user/create` | 创建用户 |
| `/api/v1/user/list` | 查询用户 |
| `/api/v1/user/update` | 更新用户 |
| `/api/v1/tunnel/list` | 列出隧道 |
| `/api/v1/tunnel/user/assign` | 分配隧道 |
| `/api/v1/tunnel/user/update` | 更新隧道绑定 |
| `/api/v1/tunnel/user/remove` | 移除隧道绑定 |
| `/api/v1/tunnel/user/list` | 列出用户隧道 |

## 故障排查

### 服务开通失败

1. 检查 WHMCS 模块日志 (Utilities > Logs > Module Log)
2. 验证 FLVX 管理员凭据是否正确
3. 确认隧道ID是否有效
4. 检查 FLVX 面板是否可访问

### 流量统计不准确

模块通过定时任务更新流量使用量，确保 WHMCS 的 **Usage Update** 任务已启用。

### 客户端显示错误

检查服务自定义字段是否正确填充。

## 文件结构

```
flvx/
├── flvx.php              # 主模块文件
├── lib/
│   └── FlvxApiClient.php # API 客户端类
└── templates/
    └── overview.tpl      # 客户端模板
```

## 版本历史

### v1.0.0
- 初始版本
- 支持用户创建、隧道分配
- 流量配额管理
- 客户端界面展示

## 许可证

本模块遵循 Apache License 2.0

## 支持

如有问题，请访问 [FLVX Telegram 群组](https://t.me/flvxpanel)
