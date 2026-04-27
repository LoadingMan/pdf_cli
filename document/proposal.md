# pdf-cli 面向 agent 的 CLI 方案

> 本文件是 pdf-cli "为什么是这个形状"的顶层说明。其它 [document/](.) 内的文件回答"是什么 / 怎么用"，本文件回答"为什么"。新人接手这个项目时应当先读这一份。

## 1. 目标与受众

pdf-cli 的最终调用方是 **agent**，不是终端用户。具体讲：

- **主要场景**：Claude Code、Claude Agent SDK、MCP client、用户自己写的 LangChain / OpenAI tool-use pipeline
- **次要场景**：脚本化批处理（CI、定时任务）
- **非目标**：人类在终端里手动敲命令、shell 自动补全的优雅体验、彩色 TTY 渲染

设计上的取舍随之而变：可读性让位于**可解析性**，灵活性让位于**可预测性**，简洁让位于**契约的稳定**。

## 2. 核心原则

### P1. 退出码必须可决策

agent 看到退出码就要能决定下一步动作（继续 / 重试 / 重登 / 改参数 / 放弃）。不应当需要解析 stderr 文本才能分类错误。

### P2. 出错时必须有结构化信号

stderr 必须能被一行 `json.parse` 拿到完整字段；即使忘了 `--format json`，也要有可正则解析的 fallback。

### P3. CLI 不揣测 agent 的策略

- 不内置自动重试（除按设计本应轮询的命令）
- 不内置降级路径
- 不假设 agent 想要哪种输出顺序

agent 的决策权完全保留在 agent 这一层。

### P4. 契约比实现稳定

退出码、错误 type 字符串、envelope 字段名、details 键名是**契约**，一旦发布就不能在 minor 版本里改义。message 文案不是契约。

### P5. agent 和人类共用一份 CLI

不做"agent 模式"和"人类模式"两套命令。同一个二进制，靠 `--format` 一个 flag 切换输出风格。这避免维护两套命令树，也让 agent 实现者可以用人类模式 debug。

## 3. 设计要点

### 3.1 退出码契约

14 个退出码，按 agent 的决策路径分组：

| 范围 | 含义类别 | agent 动作 |
|---|---|---|
| 0 | 成功 | 继续 |
| 1 | 未知（兜底） | 上报、停止 |
| 2 | 用法 | 修命令 |
| 3, 4 | 本地参数/配置 | 修参数或环境 |
| 5 | 鉴权 | 重登 |
| 6, 7 | 权限/额度 | 人工或降级 |
| 8, 9 | 资源/状态 | 改用查询 |
| 10, 11, 12, 13 | 暂时性失败 | 退避重试 |
| 14 | 任务终态失败 | 看 details，决定整链路重试 |
| 20 | CLI 自身 bug | 上报、停止 |

**关键决策**：

- **退出码 1 不再是 APIError**，改成"未分类兜底"，与 Unix 习惯对齐（1 = 通用失败、2 = 用法错误）。这是破坏性变更，老的退出码语义已废弃。
- **拆分 5 / 6 / 7**：登录、权限、额度三件事 agent 的处理动作完全不同，必须分开。
- **引入 14 (task_failed)**：把"命令执行成功 + 它告诉你之前发起的任务挂了"和"命令本身失败"分开。原来全部混在 1 里，agent 没法靠退出码判断 status --wait 的结局。
- **10 / 11 / 12 / 13 都标为 retryable**：agent 的退避策略可以差异化，但判断"是否重试"只看一个 bool 字段。

完整码表见 [conventions.md §1](conventions.md)，每个码的来源与处理动作见 [errors.md](errors.md)。

### 3.2 错误的结构化输出

**JSON envelope**（`--format json` 时写到 stderr，单行）：

```json
{"ok":false,"exit_code":14,"type":"task_failed","message":"翻译任务终止: status=fail reason=ocr_error","hint":"查看 history 或重新发起 upload→start","retryable":false,"details":{"fail_reason":"ocr_error","status":"fail","task_id":"abc-123"}}
```

**机器可读尾巴**（默认 pretty 模式，写到 stderr 末行）：

```
# pdf-cli error type=task_failed code=14 retryable=false fail_reason=ocr_error status=fail task_id=abc-123
```

**关键决策**：

- **envelope 即使在命令解析失败时也要写**：未知子命令、未知 flag、必填参数缺失这些在 cobra 的 PersistentPreRun 之前就报错的场景，靠 [cmd/root.go](../cmd/root.go) `Execute` 提前从 argv 嗅探 `--format` 来保证。
- **pretty 模式也带尾巴**：agent 即使忘加 `--format json`，仍能从尾巴正则解析。这是面向"忘记带 flag 的 agent"的安全网。
- **details 是结构化键值袋**：让 agent 拿 task_id / query_key / fail_reason 这种动作所需的标识符时不用 grep message。键 snake_case、值字符串、按字典序排列（pretty 尾巴里），保证可正则。

详见 [conventions.md §2](conventions.md)。

### 3.3 错误分类的两层映射

错误从源头到 agent 经过两层映射：

```
HTTP 层               业务码层              CLIError
─────────────         ─────────────       ───——──────
401             ──┐                       ┌── auth (5)
402             ──┤                       ├── quota_exhausted (7)
403             ──┤                       ├── permission (6)
404             ──┤    后端 code 4xx       ├── not_found (8)
409             ──┼──  message 关键字 ─-─→ ├── conflict (9)
429             ──┤    "未登录" "额度"      ├── rate_limited (10)
5xx             ──┤    "权限" "参数"       ├── server_error (13)
timeout         ──┤                       ├── timeout (12)
connect refuse  ──┘                       └── network (11)
```

第一层在 [internal/client/client.go](../internal/client/client.go) 的 `classifyHTTPStatus` / `classifyTransportError`，靠 HTTP 状态码做硬映射。第二层 `classifyBusinessError` 处理 HTTP 200 + 后端 `code != 1` 的情况，先做关键字匹配（`未登录` / `额度` / `权限` / `参数` / `not found` / `forbidden` / `quota` 等），匹配不到再看业务码本身是不是 HTTP-shaped 数字（4xx → invalid_argument，等等），都不命中才落到 unknown (1)。

**为什么这样切**：后端把许多语义错误塞在同一个 envelope 里，光看 HTTP 状态分不出来。关键字匹配是务实之选——准确率不是 100%，但只要能挑出三大类（auth / quota / permission）agent 的体验就有质的提升。剩下未识别的进 unknown，agent 看到 unknown 会上报，督促我们增量补关键字。

### 3.4 异步任务的退出码语义

`translate status` 和 `tools job status` 这类查询命令的退出码语义：

| 任务状态 | 退出码 |
|---|---|
| processing / pending | 0（仍在进行） |
| done / SUCCESS | 0（成功完成） |
| fail / cancel / FAILURE | **14** |

**关键决策**：把"任务挂了"和"中间态查询"用同一个退出码（0 vs 14）区分，**不需要 agent 解析 stdout 的 status 字段**就能判断终局。

`--wait` 模式下，CLI 内部轮询直到终态。agent 调一次 `status --wait`，靠退出码就知道结果——0 = 可以下载，14 = 整链路要重做。这是"CLI 不揣测 agent"原则的一个例外，因为"轮询直到终态"是命令的核心功能，不是隐藏的重试。

详见 [async-flows.md](async-flows.md)。

### 3.5 不内置重试

CLI 只在 `status --wait` 这一类按设计应当轮询的场景里做内部循环。**所有其它场景的重试都由 agent 决定**。

**为什么**：

- **双层重试时长爆炸**：agent 自己也在重试，CLI 再重试一遍会把超时预算吃光。
- **幂等性边界模糊**：`upload` 不幂等，自动重试会留孤儿文件。给每个命令标 idempotent 是一个不小的工程，且会让 CLI 的状态空间变大。
- **决策权错位**：quota 耗尽时 CLI 不该重试，但 agent 可能想"切到另一个账号再试"。这是 agent 的策略，不是 CLI 的策略。

[conventions.md §4](conventions.md) 给出 agent 应当遵循的退避约定。

### 3.6 命令解析层的统一

cobra 默认对几种"用法错误"的处理是不一致的：

- 未知子命令在父命令是 group 时，**会打 help 然后 exit 0**（不是 error）
- 未知 flag 走 `FlagErrorFunc`，但这个 hook 不会自动传播到子命令
- 必填 flag 缺失也走 `FlagErrorFunc`，但错误类型是 cobra 自带的

[cmd/root.go](../cmd/root.go) 的 `applyFlagErrorFunc` 统一处理这三件事：

- 递归给所有命令装 `FlagErrorFunc`，把 cobra 错误重映射为 `UsageError` / `ParamError`
- 给所有有子命令但没 RunE 的父节点装一个 `parentGroupRunE`，遇到未知子命令时退出 2
- 静默 cobra 自带的 usage 打印，避免污染 stderr 让 envelope 不可解析

`Execute` 还在 cobra 解析之前嗅探 `--format json`，让命令解析失败也能产出 envelope。

## 4. 实现位置一览

| 关注点 | 文件 |
|---|---|
| 退出码常量、type 字符串、CLIError 结构、构造函数、envelope/trailer 输出 | [internal/errors/errors.go](../internal/errors/errors.go) |
| HTTP 状态 / 后端业务码 / transport 错误分类 | [internal/client/client.go](../internal/client/client.go) |
| 命令解析层统一映射、`--format json` 提前嗅探 | [cmd/root.go](../cmd/root.go) |
| `requireAuth` 经统一 Handle | [cmd/auth.go](../cmd/auth.go) |
| translate status 终态映射到 exit 14 + details | [cmd/translate.go](../cmd/translate.go) |
| tools job status / poll 终态映射到 exit 14 / 12 + details | [cmd/tools.go](../cmd/tools.go), [cmd/tools_extra.go](../cmd/tools_extra.go) |
| 单元测试（envelope 解析、trailer 排序稳定性、details 懒分配） | [internal/errors/errors_test.go](../internal/errors/errors_test.go) |

## 5. 与 skills/ 的分工

[skills/](../skills/) 和 [document/](.) 各管一半，**不重复**：

| | skills/ | document/ |
|---|---|---|
| 受众 | Claude Code 的 Skill 加载机制（也可被任意 LLM 读取） | 通用 agent / 集成方 / 维护者 |
| 内容 | 每条命令的 flag / 参数 / 输出 / 单条命令示例 | 错误码语义、异步链路、API 映射、端到端示例、本设计方案 |
| 风格 | 命令式、强约束（"MUST"、"CRITICAL"） | 叙述+表格、解释 why |
| 文件粒度 | 一个 SKILL.md + 每条命令一份 reference | 一个 topic 一份文件 |

agent 实现者通常只读 [document/](.)；Claude Code 用户用 skills/ 自动加载。两者交叉引用，但内容不复制。

## 6. 不在本方案范围内

以下东西**故意没做**：

- **`--format json` stdout schema 的稳定化**：当前直接透传后端 `data` 字段，后端升级会影响 stdout。短期靠 agent 一侧宽松解析。长期需要为每个命令定一份 stable schema 和版本号。
- **彩色 TTY 输出**：人类友好的进度条、颜色等。和 agent 解析需求互斥。
- **shell completion**：人类用得多，agent 用不上。
- **i18n**：message 当前是中文；envelope 字段名和 type 字符串都是英文，agent 不依赖 message 文案。i18n 只影响人类可读部分。
- **CLI 内置重试**：见 §3.5。
- **stdin 管道作为输入源**：当前都是 `--file <path>`。stdin 模式可以增加但不影响契约。

## 7. 已知局限与后续改进

按优先级排序：

1. **stdout JSON schema 不稳定**：见 §6 第 1 项。如果 agent 集成方反馈频繁因后端升级而坏，需要在 CLI 这一层加适配。
2. **`translate status` 对不存在的 task-id 返回 `{}` + exit 0**：是后端透传，agent 必须检查 stdout 而不是只看退出码。修复需要在 CLI 这一层判断"空数据 + 不在 history 中" → exit 8。
3. **business code 关键字匹配可能漏识别**：agent 看到 type=unknown / exit 1 时上报，我们补关键字。这是预期的迭代路径，不是 bug。
4. **tools 高层命令没有 `--no-wait` flag**：agent 想自己管 query-key 时只能从超时 envelope 的 `details.query_key` 拿。给所有高层 tools 命令加一个 `--async` flag 直接返回 query-key 是个明确的扩展点。
5. **`auth login` 需要交互式输入密码**：agent 集成场景下没法自动登录。可以考虑加 `--password-stdin` 或 `--password-env` 让 agent 能在受控环境下注入。
6. **`internal/errors` 还保留 `APIError` 和几个 legacy 别名**（`ExitAPIError` 等）：cmd/ 下已无引用，下一次清理时可以删除。
7. **没有 build/CI 集成**：`go test ./internal/errors/` 跑通，但没接 CI。建议加一条 GitHub Actions（或同等）跑 `go vet ./... && go test ./...`，并在退出码 / envelope 字段变更时人工 review。

## 8. 决策日志（关键二选一）

| 决策 | 选择 | 否决项 | 理由 |
|---|---|---|---|
| 退出码粒度 | 14 个细分码 | 5 个粗码（成功/参数/鉴权/暂时/致命） | 多 5 个码换零文本解析，agent ROI 高 |
| 1 号码重义 | 直接改，破坏性变更 | 兼容老版本 | CLI 还无外部 shell 脚本依赖，早改成本最低 |
| stderr 错误格式 | JSON envelope + pretty trailer 双轨 | 只 JSON 或只 pretty | 双轨成本低、覆盖"忘 flag" 边角 |
| details 字段 | 加结构化键值袋 | 只放在 message 里靠 grep | 文案稳定性差，i18n 直接坏 |
| 任务失败的退出码 | exit 14 (TaskFailed) | exit 0 + 看 stdout status 字段 | agent 不需要解析 stdout 即可决策 |
| 重试机制 | CLI 不做（除 status --wait） | CLI 内置 `--retry` | 双层重试爆炸 + 幂等性边界 |
| 命令分组 | 与 skills/ 文件结构对齐 | 单一 user-guide.md | translate + tools 太大，单文件难维护 |
| 与 skills/ 的关系 | 并列、分工、不重复（方案 C） | A：document 是 source；B：双份重叠 | A 需要写生成器，B 维护成本高，C 各补各的短板 |

---

**版本**：本方案对应代码状态见 [internal/errors/errors.go](../internal/errors/errors.go) 的退出码常量定义。任何对契约的破坏性变更应当在本文件 §3 留下决策记录。
