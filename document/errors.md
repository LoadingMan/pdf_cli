# errors

每个退出码的来源、识别方式、agent 处理动作。本文件是 [conventions.md](conventions.md) §1 的展开。

> **版本说明**：v0.x → 当前版本退出码语义已整体重排。早期版本中 `1=APIError, 2=AuthError, 3=ParamError, 4=ConfigError, 5=NetError, 6=Internal` 已废弃，不与本表兼容。

## 0 — success

命令成功完成。stdout 含命令产物。stderr 通常为空（部分命令会在 stderr 写进度信息如 `正在上传文件...`，agent 可以忽略或将 stderr 作为日志收集）。

## 1 — unknown

未分类错误的兜底。**理论上不应在生产中出现**。如果 agent 看到 exit 1：

- 收集 stderr 全文 + 命令行作为 bug 报告
- 不要重试
- 考虑作为 internal 上报路径处理

实践中可能出现 exit 1 的情形：后端返回了一个我们没识别的业务码 + 没有匹配关键字的消息。

## 2 — usage

CLI 用法层面的错误，发生在命令解析阶段。

来源：
- 未知子命令：`pdf-cli notacommand`
- 未知 flag：`pdf-cli translate start --bogus`
- 必填 flag 通过 `cobra.MarkFlagRequired` 声明但写法不规范（这种实际是少见的边界情况）

agent 处理：
- **不要重试**
- 命令是 hard-coded 错的，回头检查命令构造逻辑
- 如果命令是从模型生成的，把退出码 + message 反馈给模型让它自纠

`message` 字段含 cobra 的原始错误文本，agent 可以用关键字 `unknown command` / `unknown flag` 判断细类。

## 3 — invalid_argument

参数语义错误。命令解析成功了，但参数值在 CLI 自身校验或后端校验时被拒。

来源：
- CLI 本地校验：`return clierr.ParamError(...)`，例如缺 `--task-id`、文件路径不存在
- 后端业务码 4xx 或返回消息含"参数"/"invalid argument"/"bad request"

agent 处理：
- **不要重试**（重试会拿到同样的错误）
- 如果是 ID 类参数错误，先调 `translate history` / `tools job status` 确认正确的 ID
- 如果是文件路径错误，确认本地文件确实存在
- 修参数后才重发

## 4 — config

CLI 本地配置或磁盘问题。

来源：
- `~/.config/pdf-cli/config.json` 写入失败（权限/磁盘满）
- token 持久化失败
- device-id 生成失败

agent 处理：
- 通常需要人工介入
- 检查 `~/.config/pdf-cli/` 目录权限
- 检查磁盘空间

## 5 — auth

未登录或 token 失效。

来源：
- 未登录调用了必须登录的命令（CLI 本地校验，requireAuth）
- 后端返回 HTTP 401
- 后端返回业务消息含"未登录"/"请登录"/"token"/"unauthorized"

agent 处理：
- 调用 `pdf-cli auth login --email <email>` 重新登录
- 登录后重试**原命令一次**（避免无限循环）
- 仍失败则停止并上报

注意：登录命令本身需要交互式输入密码。agent 集成场景下，登录通常应在会话开始前由人工或预配置完成；agent 在运行时遇到 exit 5 时，最稳的做法是停止并请求人工介入，而不是尝试自动登录。

## 6 — permission

已登录但被服务端拒绝。

来源：
- 后端返回 HTTP 403
- 后端业务消息含"权限"/"会员"/"forbidden"

典型场景：
- 调用了 VIP 专属命令但当前账号是普通会员
- 调用了某个未为该账号开通的功能

agent 处理：
- **不要自动重试**
- 调 `pdf-cli member info` 和 `pdf-cli member rights` 查权限详情
- 提示用户升级或换号
- 如果任务可降级（例如 `translate upload` → `translate upload --free`），可以尝试降级后重试

## 7 — quota_exhausted

额度耗尽。

来源：
- 后端返回 HTTP 402
- 后端业务消息含"额度"/"余额"/"次数不足"/"quota"/"insufficient"

agent 处理：
- **不要自动重试**
- 调 `pdf-cli user profile` 查剩余额度
- 调 `pdf-cli member pricing` 查充值方案
- 提示用户充值或等待周期重置
- 同 6：可尝试免费降级路径

## 8 — not_found

资源不存在。

来源：
- 后端返回 HTTP 404
- 后端业务消息含"不存在"/"未找到"/"not found"
- `translate download` 所有下载源均失败

典型场景：
- task-id / file-key / record-id 拼错或已过期
- 任务被另一个进程取消
- 后端清理了过期的临时文件

agent 处理：
- **不要重试**
- 用 `translate history` 或 `tools job status` 验证 ID 是否正确
- 如果 ID 来自上一步命令的输出，检查输出解析是否正确
- 任务确实丢失则重新走 upload→start

## 9 — conflict

状态冲突。

来源：
- 后端返回 HTTP 409
- 后端业务消息含"已存在"/"正在"/"conflict"/"already"

典型场景：
- 对一个已 done 的任务发起 `translate continue`
- 对一个进行中的任务发起 `translate start`

agent 处理：
- **不要重试同一命令**
- 调 `translate status` 看任务真实状态
- 根据状态决定下一步：done → 直接 download；processing → 等待；fail → 重新发起

## 10 — rate_limited

被限流。

来源：HTTP 429。

agent 处理：
- 退避后重试，起始 5s，每次 ×2，最多 5 次
- 如果 stderr envelope 含 `retry_after_ms`，使用该值
- 多个 agent 并发调用同一账号时容易触发，考虑串行化

## 11 — network

网络层错误。

来源：
- DNS 解析失败
- TCP 连接被拒
- TLS 握手失败
- 读响应体中断

agent 处理：
- 立即重试 1 次
- 仍失败则退避 1s → 2s → 4s，最多 3 次
- 持续失败考虑是否需要切换网络环境

## 12 — timeout

客户端超时。

来源：HTTP 客户端超时（默认 base API 120s，tool API 300s，文件上传 5–10 分钟）；以及 tools 异步任务的整体轮询超时（240s）。

`details` 字段：

- `query_key`（仅 tools 命令）：超时时任务通常仍在后端执行，agent 可用此 key 通过 `tools job status --query-key <key>` 继续轮询，无需重新发起任务

agent 处理：
- 重试 1 次
- 如果命令支持自定义超时 flag，第二次重试可加大
- tools 命令收到 exit 12 + `details.query_key` 时，**不要重发命令**，而是切换到异步轮询路径
- 注意：超时不代表请求未到达后端，**对非幂等命令**先用 status/history 确认状态再决定是否重试

## 13 — server_error

后端 5xx。

来源：HTTP 500/502/503/504。

agent 处理：
- 退避重试，起始 2s，每次 ×2，最多 3 次
- 持续 5xx 是后端问题，停止并上报

## 14 — task_failed

异步任务在终态被标记为失败。**与"命令失败"是不同的概念**——命令本身成功执行了，是它告诉 agent "你之前发起的任务挂了"。

来源：
- `translate status` 拿到 status ∈ {fail, cancel}
- `tools job status` 拿到 state = FAILURE
- `tools merge` / `tools convert` 等高层命令在内部轮询时拿到 FAILURE
- `translate text` SSE 流中拿到 `event: [ERROR]`
- `translate download` 在轮询过程中遇到任务进入 failed 状态

字段语义：
- `message` 含失败原因（如果后端给出）
- `hint` 通常指向 `history` 或重新发起的建议
- `details` 含结构化上下文：
  - translate 链路：`task_id`、`status`（fail 或 cancel）、`fail_reason`（若有）
  - tools 链路：`query_key`

agent 处理：
- **不要直接重试 status 命令**——状态不会自己变好
- 读 message 判断失败类型：模型失败可重试 upload→start；文件不支持需要换文件
- 根据情况决定是否重新发起整条工作流

注意：`translate download` / `tools job download` 在任务状态为 fail 时，多数情况下后端会返回 404，agent 会收到 exit 8。仅当 download 命令内部的轮询路径捕获到 FAILURE 状态时才会退出 14。两种处理逻辑相同：先调 status 或 history 看具体原因。

## 20 — internal

CLI 自身的 bug。

来源：
- panic（被 runtime 捕获并 marshal）
- 响应解析失败（JSON unmarshal 错误，正常情况下不应发生）
- 不可达分支
- 任何到达 `errors.Handle` 的非 `*CLIError` 错误（不应发生）

agent 处理：
- **永远不要重试**
- 收集 stderr + 命令行 + 时间戳作为 bug 报告
- 上报后停止，不要尝试任何 fallback

## 处理路径速查

| agent 收到的退出码 | 第一反应 |
|---|---|
| 0 | 解析 stdout，继续下一步 |
| 1, 20 | 上报、停止 |
| 2 | 修命令 |
| 3, 8, 9 | 修参数或换查询命令 |
| 4 | 提示人工介入（磁盘/权限） |
| 5 | 提示人工登录 |
| 6, 7 | 提示人工充值/升级，或尝试降级路径 |
| 10, 11, 12, 13 | 退避重试 |
| 14 | 看 message，决定是否重新发起整条工作流 |
