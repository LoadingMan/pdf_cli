# 翻译流程节点详细说明

每个节点对应原流程图的一个决策菱形或操作矩形，给出：触发条件、命令、输入解析、错误分支、后续节点。

---

## N1 · 检查登录态

**何时执行**：流程开始的第一步，总是执行。

**命令**：
```bash
pdf-cli auth status
```

**判定**：
- exit code == 0 → 已登录 → 进入 N3a
- exit code != 0 → 未登录 → 进入 N3b

**注意**：不要解析 stdout 文本（中文输出）。只看 exit code。

---

## N2 · 检查会员态

**何时执行**：N1=已登录 且 N3a=否（走高级路径）。

**命令**：
```bash
pdf-cli user profile --format json
```

**判定**：解析 JSON，取 `vipLevel`：
- `vipLevel > 0` → 会员 → N5-S1 前打印"两组引擎都可用"提示
- `vipLevel == 0` 或字段缺失 → 普通用户 → N5-S1 前打印"建议选普通"警告

**N2 不影响 N5 流程**：无论会员态，N5-S1 都展示，让用户主动选择引擎类型；vipLevel 只用于决定 stdout 提示文案。

**错误分支**：
- 接口失败（network/auth）→ 保守视为非会员，继续流程，stderr 打印警告

---

## N3a · 已登录：免费 or 高级

**何时执行**：N1=已登录。

**跳过条件**：用户原话已表明意图：
- "免费翻译…" / "用免费的翻" → free=true，跳过本节点
- "高级翻译…" / "高质量翻译" → free=false，跳过本节点

**AskUserQuestion**：见 [ask-patterns.md#n3a](ask-patterns.md#n3a-免费还是高级已登录)

**分支**：
- 选"免费" → N4（选语言）→ `upload --free --to <lang>`
- 选"高级" → N2 → N4 → N5 → N6 → `upload --advanced …`

**额度预检**：`upload` 命令内部已自动调 `user/config/homepage` 做预检（参考 `pdf-translate` skill 的"用户限制条件预检"节）。agent 不需要手动查；遇到 `quota` 错误时按错误处理表上报即可。

---

## N3b · 未登录：是否使用高级翻译

**何时执行**：N1=未登录。

**跳过条件**：
- 用户原话 "免费翻译…" / "游客翻译" → 跳过，advanced=false
- 用户原话 "高级翻译…" → 直接进入"是"分支

**AskUserQuestion**：见 [ask-patterns.md#n3b](ask-patterns.md#n3b-未登录-是否使用高级翻译)

**分支**：
- 选"是（高级）" → 输出登录提示并**终止流程**：
  ```
  高级翻译需要登录账号。请先运行：

      pdf-cli auth login --email you@example.com

  登录完成后再次发起翻译即可。
  ```
  **不要**自己调 `auth login`（需要邮箱+验证码交互，由用户在 TTY 完成）。

- 选"否（免费）" → 提示游客限制（见下），→ N4 → `upload --free --to <lang>`

**游客限制提示文本**（在选语言前打印一次）：
```
游客模式：
  • 译文不会下载到本地，只在翻译完成后控制台打印
  • 单文件大小受 freeTransMaxFileSize 限制
  • 每日免费次数受 freeTransLimitNumPerDay 限制
```

---

## N4 · 选择目标语言

**何时执行**：所有翻译路径都需要（除非用户原话已指定）。

**跳过条件**：用户原话提到目标语言（注意映射到实际代码）：
- "翻成日文/日语" → `ja`
- "翻成中文/简体" → `zh-CN`（**不是** `zh`）
- "翻成繁体中文" → `zh-TW`
- "翻成英文" → `en`
- "翻成韩文" → `ko`
- 其他显式语言名 → 用拉取的 langList 模糊匹配 `name`

跳过时仍建议 stdout 打印一行 `→ 目标语言: 日语 (ja)` 让用户确认。

**命令（拉取列表）**：
```bash
pdf-cli translate languages --format json
```

**输入解析**：响应形如 `{"langList": [{"code":"zh-CN","name":"简体中文"}, ...]}`，API 已按热度排序。

**两阶段交互**（受 AskUserQuestion 每问 ≤4 选项的工具限制；不依赖工具自动 Other，把入口显式放进 options）：
- **S1**：3 个语言 + "更多语言…" = `langList[0..2]` + 1 槽位入口（zh-CN / en / ja / 更多语言…）
- **S2**（用户选"更多语言…"时）：3 个语言 + "手动输入语言代码" = `langList[3..5]` + 1 槽位入口（ko / zh-TW / ru / 手动输入语言代码）
- **S2 选"手动输入语言代码"**：agent 不再调 AskUserQuestion，stdout 提示用户用自由文本回复语言代码或名称，并告知可运行 `pdf-cli translate languages` 查询完整列表

详细模板见 [ask-patterns.md#n4](ask-patterns.md#n4-选择目标语言两阶段)。

**输入验证**（用户文本回复或原话已说语言时）：
1. 拉 `pdf-cli translate languages --format json`
2. 精确匹配 `code` → 模糊匹配 `name`
3. 命中 → 用对应 `code` 传给 `--to`，stdout 打印 `→ 目标语言: <name> (<code>)` 让用户确认
4. **未命中** → 打印 `你输入的「<input>」不在支持的语言列表里。请先运行 pdf-cli translate languages 查看完整列表，找到对应代码后再告诉我。` 然后**终止本次流程**，不要循环追问

**AskUserQuestion**：见 [ask-patterns.md#n4](ask-patterns.md#n4-选择目标语言)

---

## N5 · 选择翻译引擎

**何时执行**：走高级路径（N3a=否 或 用户原话指定高级）。

**跳过条件**：用户原话指定引擎：
- "用 google 翻译" → engine=google
- "用 deepl/openai/…" → 对应代码

**命令（拉取列表）**：
```bash
pdf-cli translate engines --format json
```

**响应解析**：返回 `{"<engineKey>": {"engineId": <int>, "engineName":"chatgpt-4.1", "engineShowName":"...", "highLevelFlag": 0|1, "showFlag": 0|1, "tokenCostRatio": "4"}, ...}`

完整字段含义见 [`../pdf-translate/references/pdf-translate-engines.md`](../../pdf-translate/references/pdf-translate-engines.md)。

**两阶段交互**（受 AskUserQuestion 每问 ≤4 选项限制）：

- **S1**：2 选项 = 普通引擎 / 高级引擎
  - **始终展示**，会员/非会员都问，不能因会员态自动跳过——会员选普通也合理（倍率小、成本低）
  - 会员在打开 S1 前 stdout 加一行说明：两组都可用，普通便宜，高级质量高
  - 非会员在打开 S1 前 stdout 加一行警告：选高级会被后端 quota 拦截 + 普通用户额度（每日 2 次 / 每次 600 字符）
- **S2**：根据 S1 选择，从 `engines` 过滤 `showFlag==1` + `highLevelFlag` 匹配，按 `sort` 升序取**前 3 个**作为引擎 options，第 4 个槽位放显式 "手动输入引擎名称"
  - label 形如 `<engineShowName> · <tokenCostRatio>x`
  - description 冗余写 `engineName=<x>` 便于反推
- **S2 选"手动输入引擎名称"**：agent 改用文本提示，让用户回复 engineName，并告知可运行 `pdf-cli translate engines` 查询

详细模板见 [ask-patterns.md#n5](ask-patterns.md#n5-选择翻译引擎两阶段)。

**输入验证**（用户文本回复或原话已说引擎时）：
1. 拉 `pdf-cli translate engines --format json` 过滤 `showFlag==1`
2. 精确匹配 `engineName` → 不区分大小写包含匹配 `engineShowName`
3. 命中 → 用 `engineName` 传给 `--engine`，stdout 打印 `→ 引擎: <showName> · <ratio>x`
   - 若 `highLevelFlag` 与 S1 选的类型不一致 → 打印纠偏提示后**按引擎实际类型走**（engineName 匹配的信号比类型选择更强）
4. **未命中** → 打印 `你输入的「<input>」不在支持的引擎列表里。请先运行 pdf-cli translate engines 查看完整列表，找到 engineName 后再告诉我。` 然后**终止本次流程**

**关键**：不要按 `sort` 全局排序——`sort` 在普通/高级两组里各自从 1 编号会重合。必须先按 S1 选择过滤，再组内按 `sort` 排。

普通用户选了 `highLevelFlag==1` 的项时，CLI 端会返回 `quota` 错误——agent 把错误转告用户即可，不要在客户端预先拦截（让用户清楚看到所有可用选项有哪些升级能解锁）。

**AskUserQuestion**：见 [ask-patterns.md#n5](ask-patterns.md#n5-选择翻译引擎)

每个选项的 label 用 `engineShowName`，传给 `--engine` 的值用 `engineName`（字符串，如 `chatgpt-4.1` / `google`）；不要用数字 engineId。

**普通用户专属警告**（在打开问题前打印一次）：
```
你不是会员，可选普通引擎；高级引擎仅会员可用。
当前限额：每日 2 次 / 每次最多 600 字符。
```

---

## N6 · 是否启用 OCR

**何时执行**：走高级路径。

**跳过条件**：用户原话提到 OCR：
- "开 OCR" / "扫描件" / "图片 PDF" → ocr=true
- "不要 OCR" / "正常 PDF" → ocr=false

**AskUserQuestion**：见 [ask-patterns.md#n6](ask-patterns.md#n6-是否启用-ocr)

**结果**：决定 `--ocr` flag 是否传给 `translate upload`。

---

## N7 · 等待翻译状态

**何时执行**：N3-N6 走完，`upload` 命令成功返回，得到 `task-id`。

**命令**：
```bash
pdf-cli translate status --task-id <task_id> --wait --format json
```

`--wait` 会阻塞直到任务终态（done/error/cancelled）。**用 `run_in_background: true`**（翻译可能需要数分钟，不阻塞主对话）。

后台启动后，告诉用户：
```
翻译已发起，task-id = <id>。预计 1-5 分钟，状态会自动更新。
你可以继续做别的事，或随时让我查看进度。
```

**进度查询**：用户问"进度怎样了"时，用 BashOutput 读后台输出。

**完成判定**：从 stdout JSON 解析 `status` 字段。

---

## N8 · 输出译文

**何时执行**：N7=done。

### N8a · 已登录用户

**AskUserQuestion**：见 [ask-patterns.md#n8a](ask-patterns.md#n8a-是否下载译文)

- 选"下载" → 
  ```bash
  pdf-cli translate download --task-id <task_id> --output ./<原文件名>_<lang>.pdf
  ```
  下载后告诉用户文件路径。
- 选"稍后" → 仅打印：
  ```
  task-id: <id>
  之后随时可运行: pdf-cli translate download --task-id <id>
  ```

### N8b · 未登录用户（游客）

**自动行为**：N7 的 `status --wait` 在 done 分支已**自动**调用 pdftotext 打印译文。

agent **无需**额外操作，只需：
- 把 status 的 stdout（含译文）转发给用户
- 若 stdout 里出现 "pdftotext 不可用" 提示，告诉用户：
  ```
  系统未安装 pdftotext，已退化为打印预览 URL：<url>
  安装方法: apt install poppler-utils
  ```

---

## arxiv 分支

**触发**：用户原话给了 arXiv ID 而非 PDF 路径。

**差异**：
- 跳过 N1/N2（arxiv 命令支持游客）
- 跳过 upload（命令内部直接下载）
- 仍然走 N4 / N5 / N6 收集参数

**命令**：
```bash
pdf-cli translate arxiv --arxiv-id <id> --to <lang> [--engine <e>] [--ocr] --format json
```

返回的 `task-id` 进入 N7。

---

## 错误处理速查

| 错误码 | 来源 | 处理 |
|--------|------|------|
| `auth` | 任何已登录命令 | 告诉用户 token 失效，运行 `pdf-cli auth login --email …` |
| `quota` | upload / start | 报告具体限额（次数/字符/文件大小），不重试 |
| `not_found` | status / download | task-id 不存在或已过期，让用户确认 ID |
| `network` | 任何请求 | 重试一次；仍失败让用户检查网络 |
| `validation` | upload | 文件不存在或格式错误，让用户核对路径 |

详细错误处理见 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。
