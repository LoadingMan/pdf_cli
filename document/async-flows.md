# async-flows

pdf-cli 的两条异步链路：translate（PDF 翻译）与 tools（PDF 工具）。本文件给出每条链路的状态机、轮询策略、终态判断规则、以及 agent 应该在哪些点退出码 / JSON 字段上做决策。

## 1. translate 异步链路

### 1.1 流程图

```
                   ┌──────────────┐
                   │ upload       │
                   │ (--free 可选)│
                   └──────┬───────┘
                          │ 输出 file-key
                          ▼
        ┌─────────────────┴──────────────────┐
        │                                    │
   ┌────▼─────┐                       ┌──────▼──────┐
   │ start    │                       │ free        │
   │(需登录)  │                       │(无需登录)   │
   └────┬─────┘                       └──────┬──────┘
        │                                    │
        └────────────┬───────────────────────┘
                     │ 输出 task-id
                     ▼
              ┌──────────────┐    pending/processing
              │ status       │◄─────────────┐
              │ (--wait 轮询)│              │
              └──────┬───────┘              │
                     │                      │
                     │ 终态                 │
              ┌──────┼─────────┬──────┐
              │      │         │      │
              ▼      ▼         ▼      ▼
            done   fail     cancel  (+ continue 可恢复 pending)
              │      │         │
              ▼      ▼         ▼
         download exit 14   exit 14
            │
        本地文件
```

### 1.2 各命令的契约

#### `translate upload --file <path> [--free]`

- **前置条件**：本地文件存在；非 `--free` 模式需登录
- **副作用**：在服务端 S3 创建一个临时文件
- **幂等**：否（每次产生新 file-key）
- **输出**：`file-key`（也叫 `tmpFileName`）。pretty 模式打印到 stdout 末尾；JSON 模式在结果对象的 `tmpFileName` 字段
- **关键退出码**：3（文件不存在）、5（需登录但未登录，仅非 free 模式；CLI 会自动降级到 free 而不是退出 5）、10/11/12/13（网络）

#### `translate start --file-key <key> --to <lang> [...]`

- **前置条件**：上一步 upload 拿到的 file-key，登录态有效
- **副作用**：在服务端创建一个翻译任务
- **幂等**：否（每次产生新 task-id）
- **输出**：`task-id`、`record-id`
- **关键退出码**：5（未登录）、6（会员等级不足）、7（额度耗尽）、3（参数错误，含未知语言/引擎）

#### `translate free --file-key <key> --to <lang>`

- 同 `start`，但不要求登录、不消耗会员额度、引擎选择受限
- **副作用**：免费翻译队列任务
- **关键退出码**：10（免费用户限流概率高）

#### `translate status --task-id <id> [--wait]`

- **前置条件**：task-id 来自 start/free 输出
- **副作用**：无（只读）
- **幂等**：是
- **输出字段**（pretty 模式 stdout）：`status`, `translateRate`, `failReason`
- **--wait 行为**：每 5s 轮询一次，直到状态进入 {done, fail, cancel} 之一
- **退出码语义**（重要）：
  - status = done → exit 0
  - status = fail → exit 14，message 含 failReason
  - status = cancel → exit 14
  - status = pending/processing → exit 0（一次性查询）；--wait 模式继续轮询
- agent 用法：调 `--format json --wait` 然后直接看退出码即可判断结局，不需要解析 stdout

#### `translate continue --task-id <id>`

- **前置条件**：任务处于 pending 或 cancel 状态
- **幂等**：否
- **典型 exit**：9（任务已 done 或正在跑）

#### `translate cancel --task-id <id>`

- **前置条件**：任务处于 processing 状态
- **幂等**：否
- **典型 exit**：9（任务已是终态）

#### `translate download --task-id <id> [-o <path>]`

- **前置条件**：任务处于 done 状态
- **副作用**：写本地文件
- **幂等**：是（同一 task 下载多次结果相同）
- **典型 exit**：8（task 不存在或 fail，没有可下载结果）

### 1.3 状态值

| 字段 | 值 | 含义 |
|---|---|---|
| `status` | `pending` | 排队中 |
| `status` | `processing` / `running` | 正在翻译 |
| `status` | `done` | 成功完成 |
| `status` | `fail` | 失败 |
| `status` | `cancel` | 已取消 |
| `translateRate` | 0–100 | 进度百分比 |
| `failReason` | string | 失败原因，仅 status=fail 时有值 |

### 1.4 agent 推荐用法

```
1. upload --free        → 取 file-key
2. free / start         → 取 task-id
3. status --wait        → 阻塞直到终态，靠退出码判定
   - exit 0  → 进入 4
   - exit 14 → 读 message，决定重发或放弃
4. download             → 取本地文件
```

或使用非阻塞轮询（agent 自己控制节奏）：

```
3. loop:
     status --format json
     parse stdout 的 status 字段
     status == done → break
     status in {fail, cancel} → 报错退出
     sleep N seconds
```

## 2. tools 异步链路

所有 PDF 工具命令（merge / convert / split / rotate / compress / watermark / extract / etc.）共享同一条异步链路。

### 2.1 流程图

```
   ┌──────────────────────────────┐
   │ upload (单文件或多文件)      │
   │ POST /core/tools/box/file/   │
   │      aws/pre/upload          │
   │ → AWS PUT 上传               │
   │ POST /core/tools/box/file/   │
   │      new/upload              │
   │ 输出 blobFileName (file-key) │
   └──────────────┬───────────────┘
                  │
                  ▼
   ┌──────────────────────────────┐
   │ submit                       │
   │ POST /core/tools/todo/operate│
   │ payload: {action, data,      │
   │           pdfToolCode}       │
   │ 输出 queryKey                │
   └──────────────┬───────────────┘
                  │
                  ▼
   ┌──────────────────────────────┐    PROCESSING
   │ poll                         │◄────────┐
   │ GET /core/tools/operate/     │         │
   │     status?queryKey=...      │         │
   │ state: SUCCESS/FAILURE/...   │         │
   └──┬───────────────────────┬───┘         │
      │ SUCCESS                │ PROCESSING │
      ▼                        └────────────┘
   ┌──────────────┐
   │ download     │
   │ GET /pdf/box │
   │     /<name>  │
   └──────────────┘
```

agent 一般不直接看到 upload/submit/poll/download 这四个步骤——`tools merge`、`tools split` 等高层命令把整条链路打包了，默认会一直轮询到任务终态。

### 2.2 命令分类

- **同步语义命令**（实际上内部走异步，但 CLI 帮 agent 轮询完才返回）：
  - `tools merge` / `tools convert` / `tools split` / `tools reorder` / `tools rotate` / `tools page extract` / `tools page delete` / `tools page-number add` / `tools compress` / `tools watermark` / `tools overlay` / `tools extract image` / `tools extract text` / `tools metadata set` / `tools metadata remove` / `tools security encrypt` / `tools security decrypt`
  - 调用一次，得到本地文件，命令成功就 exit 0
  - 失败时根据失败点退出码：12（轮询超时，约 240s 后）、14（任务 FAILURE）、6/7（鉴权/额度）

- **显式异步命令**（拿到 query-key 后由 agent 自己管理）：
  - `tools job status --query-key <key>`：查一次状态
  - `tools job download --query-key <key>`：阻塞直到 SUCCESS 然后下载

### 2.3 状态值

tools 后端使用大写状态值：

| `state` | 含义 | CLI 处理 |
|---|---|---|
| `PENDING` / `PROCESSING` / `RECEIVED` | 进行中 | 继续轮询 |
| `SUCCESS` | 成功 | 进入下载 |
| `FAILURE` | 失败 | exit 14 |

### 2.4 轮询参数

- 轮询间隔：固定 2s
- 最大轮询次数：120（即约 240s）
- 超出则 exit 12（timeout），同时打印 query-key 让 agent 后续可继续查

### 2.5 agent 处理 tools job status 的退出码

| 场景 | exit | 备注 |
|---|---|---|
| state = SUCCESS | 0 | stdout 含结果数据 |
| state = PROCESSING/PENDING | 0 | stdout 含当前状态，agent 应继续等待 |
| state = FAILURE | 14 | stderr envelope 含 failReason |

注意 SUCCESS 与 PROCESSING 都是 exit 0，区别在 stdout 的 `state` 字段。这是 status 命令的语义："任务的当前状态"，而不是"我等到了结果"。要想"等到结果"应该用 `tools job download`，它会一直轮询到 SUCCESS 才返回。

## 3. 跨链路约定

| 概念 | translate 链路 | tools 链路 |
|---|---|---|
| 上传后的本地引用 | `file-key` (= `tmpFileName`) | `blobFileName`（CLI 内部使用） |
| 任务的引用 | `task-id` (= `blobFileName` 去扩展名) | `query-key` |
| 记录的引用 | `record-id`（数字） | 无 |
| 终态成功 | `status = done` | `state = SUCCESS` |
| 终态失败 | `status ∈ {fail, cancel}` | `state = FAILURE` |
| 失败退出码 | 14 | 14 |
| 进度字段 | `translateRate`（0–100） | 无（只有状态） |

## 4. 已知坑位

1. **translate status 对不存在的 task-id 返回 `{}` 而不是 404**：后端行为，CLI 直接透传。agent 不能仅靠 exit 0 断言"任务存在"，应当检查 stdout 是否含 status 字段。
2. **translate upload 在未登录时自动降级为 free 模式**：不会退出 5。如果 agent 期望严格的会员上传，应当在调用前先 `auth status`。
3. **tools 命令的非阻塞模式**：高层命令（如 `tools merge`）没有 `--no-wait` flag。如果 agent 想自己管理 query-key，目前没有官方路径——只能解析失败 stderr 中的 query-key（exit 12 时会带）。这是当前 CLI 的局限。
4. **AWS S3 上传有独立的 10 分钟超时**：与默认 base API 的 120s 超时不同。大文件上传不要因为外层 120s 误判超时。
