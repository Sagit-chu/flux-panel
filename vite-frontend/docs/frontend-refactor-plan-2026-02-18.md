# 前端彻底重构计划（进行中）

- 分支：`frontend/refactor-audit-20260218`
- 范围：`vite-frontend/src`
- 更新时间：2026-02-19
- 当前总体进度：`99.5%`

## 1) 问题审计（已完成）

### P0（高风险，优先修复）

- [x] `src/api/network.ts` 存在 token 过期后 Promise 不 resolve 的路径（`then/catch` 里仅 `return`，调用方可能挂起）
- [x] 登录态读写逻辑分散在多个页面/布局，存在重复和不一致风险（`src/pages/index.tsx`、`src/layouts/admin.tsx`、`src/layouts/h5.tsx`、`src/pages/profile.tsx`、`src/pages/group.tsx`）
- [x] 核心页面超大文件导致可维护性差：`forward.tsx`(3263 行)、`tunnel.tsx`(2595 行)、`node.tsx`(2194 行)、`user.tsx`(1761 行)、`dashboard.tsx`(1363 行)
- [x] `any` 使用过多，类型边界不清（`src/api/index.ts`、`src/api/network.ts`、`src/pages/*`）

### P1（中风险，本次并行推进）

- [x] 多处空 `catch {}` 吞错，定位问题困难（如 `src/App.tsx`、`src/components/navbar.tsx`、`src/pages/config.tsx` 等）
- [x] 移动端/H5/WebView 判定逻辑重复（`src/App.tsx`、`src/pages/index.tsx`、`src/components/navbar.tsx`）
- [x] 菜单与权限控制在多处重复定义，布局和页面耦合偏高（`src/layouts/admin.tsx`、`src/layouts/h5.tsx`、`src/pages/profile.tsx`）
- [x] 路由保护与布局选择逻辑集中在 `src/App.tsx`，可测试性和扩展性偏弱

### P2（优化项，后续阶段）

- [x] `vite.config.ts` 生产构建配置 `minify: false`、`treeshake: false`，性能优化策略待梳理
- [x] ESLint 中 `react-hooks/exhaustive-deps` 关闭，副作用依赖约束较弱（`eslint.config.mjs`）

## 2) 彻底重构任务清单（边做边更新）

> 状态说明：`[x] 完成` / `[ ] 未开始` / `[-] 进行中`

### Phase A - 会话与权限基建

- [x] A1. 新增统一会话工具模块（token/role/name/admin 的读写与兼容逻辑）
- [x] A2. 登录页改为调用会话工具写入登录态，去重重复代码
- [x] A3. 布局与页面的管理员判断改为统一工具，移除重复逻辑
- [x] A4. `Network` 层统一 token 获取与 token 过期处理，避免悬挂 Promise

### Phase B - Hook 与状态复用

- [x] B1. 提取 `useWebViewMode`，替换 `index/navbar` 重复检测
- [x] B2. 提取 `useH5Mode`，收敛 `App` 内设备与参数判定逻辑
- [x] B3. 提取滚动复位逻辑（`H5/H5-simple`）到通用 hook

### Phase C - API 与类型边界

- [x] C1. 收紧 `src/api/network.ts` 的 `any` 边界（优先 `unknown` + 受控转换）
- [-] C2. 给高频 API 接口补全请求/响应类型（先用户、节点、隧道、转发）
- [-] C3. 统一错误消息提取策略，避免散落式字符串拼接

### Phase D - 大页面拆分（增量，不大爆炸重写）

- [-] D1. `forward.tsx` 抽离：筛选条、列表视图、批量操作、详情弹窗
- [-] D2. `tunnel.tsx` 抽离：表单区、列表区、排序区、诊断区
- [-] D3. `node.tsx` 抽离：状态区、安装命令区、排序区、WebSocket 区
- [ ] D4. `dashboard.tsx` 抽离：统计卡片、图表区、公告区、刷新逻辑

### Phase E - 体验与可访问性

- [-] E1. 统一关键交互控件的 `aria-label` / 键盘可达性
- [-] E2. 统一 loading/empty/error 三态展示组件
- [-] E3. 统一移动端断点与布局响应策略

## 15) 当前实施批次（Batch-12）

- 目标：推进 **D2/D4/E1/E2/E3** 收尾，完成表单模块接入、仪表盘增量拆分与通用能力落地
- 本批次进度：`6 / 6`

### Batch-12 明细进度

- [x] T51. `tunnel.tsx` 接入 `tunnel/form.ts`（默认值、表单校验、类型/流量展示）
- [x] T52. 修复 `limit.tsx` 构建阻断问题（移除未使用 `Spinner` 导入）
- [x] T53. `dashboard.tsx` 抽离公告区与指标卡公共组件（`dashboard/components/*`）
- [x] T54. `dashboard.tsx` 管理员判定切换为 `session` 统一工具（`getAdminFlag`）
- [x] T55. `admin.tsx` 接入 `useMobileBreakpoint` 收敛断点监听；`settings.tsx` 回填返回按钮 `aria-label`
- [x] T56. 执行 `npm run build` + `npm run lint` 校验（lint 仅剩既有 `no-console` warning）

## 3) 当前实施批次（Batch-1）

- 目标：先完成 **A1/A2/A3/A4 + B1/B2 + C1**，优先解决架构一致性和稳定性风险
- 本批次进度：`7 / 7`

### Batch-1 明细进度

- [x] T1. 新增 `session` 统一会话工具（A1）
- [x] T2. 登录页切换到会话工具（A2）
- [x] T3. `admin/h5/profile/group` 切换管理员判定工具（A3）
- [x] T4. `network` 统一 token 与过期返回（A4 + C1）
- [x] T5. 新增 `useWebViewMode` 并接入 `index/navbar`（B1）
- [x] T6. 新增 `useH5Mode` 并接入 `App`（B2）
- [x] T7. 执行 `npm run build` 与必要诊断校验

## 4) 当前实施批次（Batch-2）

- 目标：推进 **D2/D3 的排序逻辑抽离** 与 **会话 token 复用**，降低大型页面重复代码
- 本批次进度：`7 / 7`

### Batch-2 明细进度

- [x] T8. 新增通用排序存储工具（node/tunnel 共用）
- [x] T9. `tunnel.tsx` 接入排序存储工具
- [x] T10. `node.tsx` 接入排序存储工具
- [x] T11. `node.tsx` WebSocket token 改用统一会话工具
- [x] T12. 执行 `npm run build` 与必要诊断校验
- [x] T13. 新增 `useScrollTopOnPathChange` 通用 hook
- [x] T14. `h5/h5-simple` 接入滚动复位 hook 并完成构建校验

## 5) 当前实施批次（Batch-3）

- 目标：推进 **D3（node WebSocket 区域）结构化抽离**，先拆出系统信息解析逻辑
- 本批次进度：`4 / 4`

### Batch-3 明细进度

- [x] T15. 新增 `node` 系统信息解析工具模块
- [x] T16. `node.tsx` WebSocket `info` 消息处理接入解析工具
- [x] T17. 保持在线/离线状态切换与速度计算逻辑一致
- [x] T18. 执行 `npm run build` 与必要诊断校验

## 6) 外部最佳实践参考（已纳入本计划）

- React 官方：重复逻辑应抽为自定义 Hook（`reusing-logic-with-custom-hooks`）
- React 官方：共享逻辑不共享状态，跨组件共享状态应提升或集中管理
- Vite 官方：大型项目优先审计插件成本、动态导入与分块策略
- WAI-ARIA APG：导航与交互组件优先语义化与键盘可达性

## 7) 当前实施批次（Batch-4）

- 目标：继续推进 **D3（node WebSocket 区域）**，抽离连接生命周期与重连策略
- 本批次进度：`4 / 4`

### Batch-4 明细进度

- [x] T19. 新增 `useNodeRealtime` Hook（连接/重连/断开）
- [x] T20. `node.tsx` 接入 `useNodeRealtime`，移除内联连接管理代码
- [x] T21. 保留离线延迟逻辑并在页面卸载时清理离线定时器
- [x] T22. 执行 `npm run build` 与必要诊断校验

## 8) 当前实施批次（Batch-5）

- 目标：继续推进 **D3（node WebSocket 区域）**，抽离离线延迟定时器生命周期
- 本批次进度：`4 / 4`

### Batch-5 明细进度

- [x] T23. 新增 `useNodeOfflineTimers` Hook（离线延迟/清理）
- [x] T24. `node.tsx` 接入 `useNodeOfflineTimers`，移除内联定时器管理
- [x] T25. 校验状态/信息消息路径行为一致（在线切换与离线延迟）
- [x] T26. 执行 `npm run build` 与必要诊断校验

## 9) 当前实施批次（Batch-6）

- 目标：推进 **D2（tunnel 诊断区）**，抽离诊断兜底与质量评估逻辑
- 本批次进度：`4 / 4`

### Batch-6 明细进度

- [x] T27. 新增 `tunnel/diagnosis` 诊断工具模块
- [x] T28. `tunnel.tsx` 接入诊断兜底与质量评估工具
- [x] T29. 校验诊断弹窗展示逻辑与质量标签行为一致
- [x] T30. 执行 `npm run build` 与必要诊断校验

## 10) 当前实施批次（Batch-7）

- 目标：推进 **C2/C3（API 类型边界与错误消息统一）**，先覆盖高频 node/tunnel/forward 路径
- 本批次进度：`4 / 4`

### Batch-7 明细进度

- [x] T31. 新增 `api` 高频领域类型定义（node/tunnel/forward）
- [x] T32. `api/index.ts` 的高频列表接口接入类型定义
- [x] T33. 新增网络错误消息提取工具并接入 `network.ts`
- [x] T34. 执行 `npm run build` 与必要诊断校验

## 11) 当前实施批次（Batch-8）

- 目标：推进 **D1（forward 排序区）**，抽离直接模式排序初始化逻辑
- 本批次进度：`4 / 4`

### Batch-8 明细进度

- [x] T35. 新增 `forward/order` 排序工具模块
- [x] T36. `forward.tsx` 接入排序工具并复用通用存储
- [x] T37. 校验直接模式排序初始化与拖拽持久化行为一致
- [x] T38. 执行 `npm run build` 与必要诊断校验

## 12) 当前实施批次（Batch-9）

- 目标：推进 **D1（forward 诊断区）**，抽离诊断兜底与质量评估逻辑
- 本批次进度：`4 / 4`

### Batch-9 明细进度

- [x] T39. 新增 `forward/diagnosis` 诊断工具模块
- [x] T40. `forward.tsx` 接入诊断兜底与质量评估工具
- [x] T41. 校验诊断弹窗展示与质量标签行为一致
- [x] T42. 执行 `npm run build` 与必要诊断校验

## 13) 当前实施批次（Batch-10）

- 目标：推进 **D1（forward 批量操作区）**，抽离批量动作执行与反馈逻辑
- 本批次进度：`4 / 4`

### Batch-10 明细进度

- [x] T43. 新增 `forward/batch-actions` 批量操作工具模块
- [x] T44. `forward.tsx` 接入批量操作工具并移除重复处理分支
- [x] T45. 校验批量删除/启停/重下发/换隧道行为一致
- [x] T46. 执行 `npm run build` 与必要诊断校验

## 14) 当前实施批次（Batch-11）

- 目标：推进 **D1（forward 地址展示/复制区）**，抽离地址格式化与弹窗分流逻辑
- 本批次进度：`4 / 4`

### Batch-11 明细进度

- [x] T47. 新增 `forward/address` 地址工具模块
- [x] T48. `forward.tsx` 接入地址工具并移除内联格式化/分流逻辑
- [x] T49. 校验地址单项复制与多项弹窗行为一致
- [x] T50. 执行 `npm run build` 与必要诊断校验
