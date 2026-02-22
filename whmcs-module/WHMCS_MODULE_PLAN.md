# FLVX WHMCS Provisioning Module - 开发计划

**创建时间:** 2026-02-22  
**状态:** 开发完成

---

## 项目概述

### 目标
开发一个 WHMCS Provisioning Module，用于对接 FLVX 面板，实现：
- 用户自动创建和管理
- 隧道分配和配置
- 流量配额设置
- 服务到期管理
- 客户端服务详情展示

### 目标用户
- 使用 WHMCS 进行计费管理的 IDC / VPS 服务商
- 需要销售流量转发服务的商家

---

## 模块结构

```
whmcs-module/
├── flvx/
│   ├── flvx.php              # 主模块文件 (元数据 + 核心函数)
│   ├── flvxClientArea.php    # 客户端区域模板
│   ├── lib/
│   │   ├── FlvxApiClient.php # FLVX API 客户端类
│   │   └── helpers.php       # 辅助函数
│   └── templates/
│       ├── clientarea.tpl    # 客户端区域 Smarty 模板
│       ├── overview.tpl      # 概览页面
│       └── config.tpl        # 配置选项模板
├── WHMCS_MODULE_PLAN.md      # 本计划文档
└── README.md                  # 安装说明
```

---

## 开发任务清单

### 阶段一：基础架构
 [x] **1.1** 创建模块目录结构
 [x] **1.2** 实现主模块文件 `flvx.php` 元数据
 [x] **1.3** 实现 FLVX API 客户端类 `FlvxApiClient.php`
 [x] **1.4** 定义产品配置参数 (Config Options)

### 阶段二：核心功能
 [x] **2.1** 实现 `flvx_CreateAccount()` - 创建用户和分配隧道
 [x] **2.2** 实现 `flvx_SuspendAccount()` - 暂停服务
 [x] **2.3** 实现 `flvx_UnsuspendAccount()` - 恢复服务
 [x] **2.4** 实现 `flvx_TerminateAccount()` - 终止/删除服务
 [x] **2.5** 实现 `flvx_ChangePassword()` - 修改密码
 [x] **2.6** 实现 `flvx_ChangePackage()` - 升级/降级套餐

### 阶段三：客户端区域
 [x] **3.1** 实现 `flvx_ClientArea()` - 客户端区域输出
 [x] **3.2** 创建 Smarty 模板展示服务详情
 [x] **3.3** 显示流量使用情况
 [x] **3.4** 显示到期时间
 [x] **3.5** 显示隧道/节点信息

### 阶段四：管理功能
 [x] **4.1** 实现 `flvx_AdminServicesTabFields()` - 管理后台字段
 [x] **4.2** 实现 `flvx_AdminServicesTabFieldsSave()` - 保存管理字段
 [x] **4.3** 实现 `flvx_ServiceSingleSignOn()` - 单点登录

### 阶段五：测试与文档
 [x] **5.1** 编写安装文档 `README.md`
 [ ] **5.2** 测试完整生命周期 (创建→暂停→恢复→删除)
 [ ] **5.3** 测试升级/降级场景
 [x] **5.4** 错误处理和日志记录

---

## 技术规范

### FLVX API 对接

#### 认证方式
- **类型:** JWT Token
- **Header:** `Authorization: <raw_token>` (无 Bearer 前缀)
- **获取:** 调用 `/api/v1/user/login` 获取 token

#### 核心 API 端点

| 功能 | 端点 | 方法 |
|------|------|------|
| 登录获取Token | `/api/v1/user/login` | POST |
| 创建用户 | `/api/v1/user/create` | POST |
| 更新用户 | `/api/v1/user/update` | POST |
| 删除用户 | `/api/v1/user/delete` | POST |
| 用户详情 | `/api/v1/user/package` | POST |
| 列出隧道 | `/api/v1/tunnel/list` | POST |
| 分配隧道 | `/api/v1/tunnel/user/assign` | POST |
| 更新绑定 | `/api/v1/tunnel/user/update` | POST |
| 移除绑定 | `/api/v1/tunnel/user/remove` | POST |
| 列出用户隧道 | `/api/v1/tunnel/user/list` | POST |

#### API 响应格式
```json
{
  "code": 0,        // 0=成功, 非0=错误
  "msg": "操作成功",
  "data": { ... },
  "ts": 1700000000  // 时间戳(毫秒)
}
```

### WHMCS 模块规范

#### 必须实现的函数
```php
// 元数据
function flvx_MetaData() {}

// 配置选项
function flvx_ConfigOptions() {}

// 创建账户
function flvx_CreateAccount($params) {}

// 暂停账户
function flvx_SuspendAccount($params) {}

// 恢复账户
function flvx_UnsuspendAccount($params) {}

// 终止账户
function flvx_TerminateAccount($params) {}

// 客户端区域
function flvx_ClientArea($params) {}
```

#### 可选实现的函数
```php
// 修改密码
function flvx_ChangePassword($params) {}

// 升级/降级
function flvx_ChangePackage($params) {}

// 管理后台字段
function flvx_AdminServicesTabFields($params) {}

// 保存管理字段
function flvx_AdminServicesTabFieldsSave($params) {}

// 单点登录
function flvx_ServiceSingleSignOn($params) {}

// 用量统计 (用于计费)
function flvx_UsageUpdate($params) {}
```

---

## 产品配置参数设计

### 基础配置 (Config Options)

| 名称 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `traffic_quota` | number | 流量配额 (GB) | 100 |
| `max_forwards` | number | 最大转发数量 | 10 |
| `expiry_days` | number | 有效期 (天) | 30 |
| `speed_limit` | dropdown | 速度限制 | unlimited |
| `tunnel_group` | dropdown | 隧道分组 | default |

### 可配置选项 (Product Custom Fields)

| 字段名 | 类型 | 描述 |
|--------|------|------|
| `flvx_user_id` | text | FLVX 用户ID (系统自动填充) |
| `flvx_tunnel_id` | text | 分配的隧道ID |
| `flvx_user_tunnel_id` | text | UserTunnel 关联ID |

---

## 业务流程

### 创建服务流程
```
1. WHMCS 接收订单
   ↓
2. flvx_CreateAccount() 被调用
   ↓
3. 检查是否已存在用户 (通过 email 或 username 匹配)
   ↓
4. 如不存在: 调用 /api/v1/user/create 创建用户
   ↓
5. 调用 /api/v1/tunnel/user/assign 分配隧道
   ↓
6. 保存 flvx_user_id, flvx_user_tunnel_id 到服务自定义字段
   ↓
7. 返回成功
```

### 暂停服务流程
```
1. 管理员或系统触发暂停
   ↓
2. flvx_SuspendAccount() 被调用
   ↓
3. 调用 /api/v1/tunnel/user/update 设置 status=0
   ↓
4. 返回成功
```

### 恢复服务流程
```
1. 管理员或系统触发恢复
   ↓
2. flvx_UnsuspendAccount() 被调用
   ↓
3. 调用 /api/v1/tunnel/user/update 设置 status=1
   ↓
4. 返回成功
```

### 终止服务流程
```
1. 管理员或系统触发终止
   ↓
2. flvx_TerminateAccount() 被调用
   ↓
3. 调用 /api/v1/tunnel/user/remove 移除隧道绑定
   ↓
4. 可选: 调用 /api/v1/user/delete 删除用户
   ↓
5. 清除服务自定义字段
   ↓
6. 返回成功
```

---

## 错误处理

### 错误代码映射
| FLVX Code | 含义 | WHMCS 处理 |
|-----------|------|------------|
| 0 | 成功 | 继续 |
| -2 | 数据库错误 | 记录日志, 返回错误 |
| -1 | 通用错误 | 返回错误消息 |
| 401 | 认证失败 | 刷新Token重试 |
| 500 | 请求参数错误 | 记录日志, 返回错误 |

### 日志记录
```php
logModuleCall(
    'flvx',
    'CreateAccount',
    $requestData,
    $responseData,
    $processedData,
    $status
);
```

---

## 安装要求

### WHMCS 要求
- WHMCS 版本: 8.0+
- PHP 版本: 7.4+
- 扩展: cURL, JSON

### FLVX 面板要求
- FLVX 版本: 2.1.0+
- 已配置 JWT_SECRET
- 已创建至少一个隧道

### 模块配置
在 WHMCS 后台配置以下参数:
- `api_url` - FLVX 面板地址 (如 https://panel.example.com)
- `admin_username` - 管理员用户名
- `admin_password` - 管理员密码 (或使用 API Token)
- `default_tunnel_id` - 默认分配的隧道ID
- `default_speed_limit` - 默认速度限制

---

## 开发进度追踪

| 日期 | 完成任务 | 备注 |
|------|----------|------|
| 2026-02-22 | 创建计划文档 | 规划完成 |
| 2026-02-22 | 完成基础架构 | 目录结构 + API客户端 |
| 2026-02-22 | 完成核心功能 | CreateAccount, Suspend, Unsuspend, Terminate |
| 2026-02-22 | 完成可选功能 | ChangePassword, ChangePackage |
| 2026-02-22 | 完成客户端区域 | ClientArea + Smarty模板 |
| 2026-02-22 | 完成管理功能 | AdminServicesTabFields, SSO |
| 2026-02-22 | 编写安装文档 | README.md |
| 2026-02-22 | 修复模板JS bug | 防止元素不存在时报错 |

---

## 参考资料

- [WHMCS Provisioning Module Documentation](https://developers.whmcs.com/provisioning-modules/)
- [WHMCS Sample Module](https://github.com/WHMCS/sample-provisioning-module)
- [Proxmox VE for WHMCS](https://github.com/The-Network-Crew/Proxmox-VE-for-WHMCS) - 参考实现
- FLVX API 文档 - 见 AGENTS.md

---

## 更新日志

### v1.0.0
- 初始版本发布
- 支持用户创建、隧道分配、流量管理
- 客户端区域展示
- 面板API完全兼容 (已验证)
