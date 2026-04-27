# conventions

pdf-cli 的 CLI 契约。本文件描述退出码、错误结构、输出格式、重试策略。所有 agent 集成方应当先读这一份再读其他模块文档。

## 1. 退出码表

退出码是 agent 唯一不需要解析的信号。每个码对应一个明确的处理动作。

| 码 | 名称 | 含义 | retryable | agent 处理动作 |
|---|---|---|---|---|
| 0 | success | 命令成功完成 | – | 继续下一步 |
| 1 | unknown | 未分类错误（兜底，理论上不应出现） | no | 上报、停止 |
| 2 | usage | 命令用法错误：未知子命令、未知 flag | no | 不要重试，检查命令拼写 |
| 3 | invalid_argument | 参数语义错误：缺必填、值非法、文件不存在、ID 找不到 | no | 修参数后重新调用 |
| 4 | config | 本地配置/磁盘问题：config.json 写不了、token 存不下 | no | 提示用户检查权限/磁盘 |
| 5 | auth | 未登录或 token 失效（HTTP 401 / 后端"未登录"） | no | 调用 `auth login` 后重试 |
| 6 | permission | 已登录但无权限：会员等级不够、功能未开通（HTTP 403） | no | 提示升级会员或更换账号 |
| 7 | quota_exhausted | 额度/配额耗尽（HTTP 402 / 后端额度码） | no | 提示充值或等周期重置 |
| 8 | not_found | 资源不存在：task-id / file-key / record-id（HTTP 404） | no | 检查 ID 是否拼错或已过期 |
| 9 | conflict | 状态冲突：任务已在跑/已完成/已取消（HTTP 409） | no | 改用查询命令而不是重发起 |
| 10 | rate_limited | 被限流（HTTP 429） | yes | 退避后重试，起步 5s |
| 11 | network | 连接失败、DNS 失败、TLS 失败、读写中断 | yes | 立即重试 1 次，仍失败则退避 |
| 12 | timeout | 客户端超时 | yes | 立即重试，可考虑增大超时 |
| 13 | server_error | HTTP 5xx | yes | 退避重试 |
| 14 | task_failed | 异步任务在服务端被标记失败（status = failed/cancel） | no | 读取 reason 决定是否重新 upload→start |
| 20 | internal | CLI 自身 bug：panic、JSON marshal 失败、不可达分支 | no | 上报、停止 |

详见 [errors.md](errors.md)。

## 2. 错误输出格式

CLI 出错时同时写 stderr 并以非零退出码退出。stderr 的格式由 `--format` 决定，stdout 不会出现错误信息。

### 2.1 `--format json`（agent 推荐）

stderr 是**单行 JSON**（不 pretty-print，方便 agent 按行解析）：

```json
{"ok":false,"exit_code":5,"type":"auth","message":"token 已失效","hint":"请执行 pdf-cli auth login 重新登录","retryable":false,"http_status":401,"backend_code":"1001"}
```

字段约定：

| 字段 | 类型 | 必有 | 说明 |
|---|---|---|---|
| `ok` | bool | 是 | 永远 false（成功不写 envelope） |
| `exit_code` | int | 是 | 与进程退出码一致 |
| `type` | string | 是 | 见 1 节"名称"列 |
| `message` | string | 是 | 人类可读消息，可能含中文 |
| `retryable` | bool | 是 | 是否值得 agent 自动重试 |
| `hint` | string | 否 | 处理建议；可能为空 |
| `http_status` | int | 否 | 仅当源头是 HTTP 错误时存在 |
| `backend_code` | string | 否 | 仅当后端返回了业务码时存在 |
| `details` | object<string,string> | 否 | 上下文相关的结构化数据；键 snake_case；见 §2.3 |

### 2.3 details 字段

`details` 是一个字符串到字符串的小型 map，用来携带 agent 可以直接 act-on 的上下文数据，避免 agent 必须 grep `message`。键约定：

| 键 | 出现位置 | 含义 |
|---|---|---|
| `query_key` | `tools job *` 类命令的超时 / 任务失败 | 当前任务的 query-key，agent 可用它继续轮询或下载 |
| `task_id` | `translate status` 终态失败 | 翻译任务 ID |
| `status` | `translate status` 终态失败 | 终态值，目前 `fail` 或 `cancel` |
| `fail_reason` | `translate status` 终态失败 | 后端给出的失败原因（如果有） |

未来可能新增更多键。**agent 应当用 `key in obj` 判断键是否存在，对未知键宽容忽略**。

典型用法（jq）：

```bash
err=$(pdf-cli --format json tools merge --files a.pdf,b.pdf -o out.pdf 2>&1)
exit_code=$?
if [ $exit_code -eq 12 ]; then
  qk=$(echo "$err" | jq -r '.details.query_key // empty')
  if [ -n "$qk" ]; then
    pdf-cli --format json tools job download --query-key "$qk" -o out.pdf
  fi
fi
```

agent 端解析建议：用 `key in obj` 判断可选字段是否存在，不要用 `value == 0/""`。

JSON envelope 在以下情形也会写出（即使命令解析失败）：
- 未知子命令
- 未知 flag
- 必填 flag 缺失
只要 `--format json` 出现在 argv 任意位置即生效。

### 2.2 默认 pretty 模式

stderr 是两行人类可读 + 一行机器可读尾巴：

```
Error: 翻译任务终止: status=fail reason=ocr_error
Hint: 查看 history 或重新发起 upload→start
# pdf-cli error type=task_failed code=14 retryable=false fail_reason=ocr_error status=fail task_id=abc-123
```

尾巴格式：

```
# pdf-cli error type=<type> code=<code> retryable=<bool>[ http_status=<int>][ backend_code=<str>][ <detail_key>=<detail_val>...]
```

- 字段顺序固定：`type` → `code` → `retryable` → `http_status` → `backend_code` → details（按键名字典序）
- 字段间用单空格分隔，键值之间用 `=`
- 值不会出现空格（detail 值约定为 ID 类字符串）
- 解析建议：按空格 split，对每段再 split `=`，得到 key/value 对

agent 即使忘了加 `--format json` 也能从尾巴拿到结构化信号，包括 details 字段。

## 3. 标准输出格式

`--format` 取值：

| 值 | 行为 |
|---|---|
| `pretty` | 默认。表格 / key-value，方便人读 |
| `json` | stdout 输出 JSON。**当前 schema 不稳定**，agent 不应硬编码字段路径，使用前应通过 `--help` 或本仓库的 [api-mapping.md](api-mapping.md) 确认 |
| `table` | 仅部分命令支持，效果同 pretty |

`--format json` 影响 stdout（命令结果）和 stderr（错误 envelope）两端。

> **stdout JSON 的稳定性**：当前直接透传后端返回的 `data` 字段，后端 schema 升级会直接影响 stdout。agent 实现应当在自身一侧加 schema 校验或宽松解析，不要 panic on missing field。

## 4. 重试策略约定

CLI **不内置自动重试**（除 `translate status --wait` 这类按设计轮询的命令）。重试策略由 agent 实现，遵循以下约定：

### 4.1 是否重试

只看 `retryable` 字段（或退出码是否 ∈ {10, 11, 12, 13}）。其他码一律不重试。

### 4.2 退避

| 退出码 | 起始延迟 | 退避因子 | 最大重试次数 |
|---|---|---|---|
| 11 network | 1s | 2× | 3 |
| 12 timeout | 1s | 2× | 3 |
| 13 server_error | 2s | 2× | 3 |
| 10 rate_limited | 5s | 2× | 5 |

`exit 10` 时如果 stderr envelope 含 `retry_after_ms`（若服务端给出），优先使用该值。

### 4.3 命令幂等性

agent 在重试前必须确认命令幂等。下表给出已知的幂等性：

| 命令 | 幂等 | 说明 |
|---|---|---|
| `translate languages` / `translate engines` | 是 | 只读 |
| `translate status` | 是 | 只读 |
| `translate history` | 是 | 只读 |
| `auth status` | 是 | 只读 |
| `user *` (查询) | 是 | 只读 |
| `member info` / `member rights` / `member pricing` | 是 | 只读 |
| `translate upload` | **否** | 每次产生新 file-key，重试会留孤儿文件 |
| `translate start` | **否** | 每次起新任务 |
| `translate free` | **否** | 同上 |
| `translate text` | **否** | 重试会重复扣额度（如有） |
| `tools merge` / `tools convert` / etc | **否** | 每次起新任务 |

非幂等命令的建议处理：
- 如果失败发生在 CLI 本地（exit 3/4/20），重试通常安全；
- 如果失败发生在请求发出之后（exit 10/11/12/13），重试可能产生重复任务。建议先调对应 `history` / `status` 命令确认后端状态，再决定是否重发。

## 5. 全局 flag

| flag | 适用范围 | 说明 |
|---|---|---|
| `--format <pretty\|json>` | 所有命令 | 见 2、3 节 |
| `--output <path>` | 大多数下载/导出命令 | 输出文件路径 |
| `--help` / `-h` | 所有命令 | 打印帮助 |

`--format` 是 persistent flag，可以放在任意位置：`pdf-cli --format json translate status ...` 与 `pdf-cli translate --format json status ...` 等价。

## 6. 命令分类（鉴权要求）

| 类别 | 鉴权要求 | 命令 |
|---|---|---|
| 公开 | 无 | `translate languages`, `translate engines`, `translate free`, `translate arxiv*`, `translate text`, `translate upload --free`, `translate status`, `translate download`, `other *` |
| 必须登录 | token | `auth status`, `auth logout`, `translate start`, `translate continue`, `translate cancel`, `translate history`, `translate upload`（非 free）, `user *`, `member *`, `tools *` |
| 登录可选 | token 优先，可降级 | `translate upload` 在未登录时自动切换为 `--free` 模式 |

未登录调用必须登录的命令时，CLI 退出码 5 (auth)。
