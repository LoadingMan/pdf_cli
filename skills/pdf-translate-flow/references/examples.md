# 端到端示例

三个完整对话流，覆盖流程图主要分支。每个示例分两栏：左边是用户消息和 AskUserQuestion 选择，右边是 agent 的工具调用。

---

## 示例 1：游客免费翻译（最常见）

**用户**: `翻译 ./paper.pdf 翻成日文`

**agent 思考**：用户给了路径 + 目标语言（日文），跳过 N4。

```bash
# N1
$ pdf-cli auth status
# exit 1 → 未登录
```

**agent 思考**：未登录 + 用户没说"高级"，N3b 需要问。

→ AskUserQuestion (N3b)

**用户选**: 免费翻译（游客可用）

**agent**：打印游客限制提示，跳过 N4（已知 ja），直接上传。

```bash
# upload (合并 free + 上传 + 发起翻译)
$ pdf-cli translate upload --file ./paper.pdf --free --to ja --format json
# 注意：中文用 zh-CN/zh-TW，不是 zh
# → {"taskId": "abc123", "recordId": 456, ...}
```

**agent**：解析 taskId=abc123，启动后台等待。

```bash
# N7 (后台)
$ pdf-cli translate status --task-id abc123 --wait --format json
# run_in_background: true
```

**agent 回复**：
> 已上传并发起免费翻译，task-id = abc123。预计 1-3 分钟。
> 完成后会自动在控制台打印日文译文。

**3 分钟后任务完成**，后台进程 stdout 包含译文文本（pdftotext 提取的纯文本）。

**agent**：把译文 stdout 转发给用户，结束。

---

## 示例 2：会员高级翻译 + OCR

**用户**: `高级翻译 ./scan.pdf 翻成中文 开 OCR`

**agent 思考**：用户给了：路径 + 高级 + 中文 + OCR。N3a/N4/N6 全部跳过。

```bash
# N1
$ pdf-cli auth status   # exit 0 → 已登录

# N2 (因为走高级)
$ pdf-cli user profile --format json
# → {"vipLevel": 2, ...}

# N5 数据
$ pdf-cli translate engines --format json
# → {"google": {...}, "deepl": {...}, "openai-premium": {...}, ...}
```

**agent 思考**：用户没指定引擎，需要问 N5。会员可见所有 showFlag=1 的引擎。

→ AskUserQuestion (N5)

**用户选**: OpenAI Premium

**agent**：所有参数齐全。

```bash
$ pdf-cli translate upload \
    --file ./scan.pdf \
    --advanced \
    --to zh-CN \
    --engine chatgpt-4.1 \
    --ocr \
    --format json
# → {"taskId": "xyz789", ...}

# N7
$ pdf-cli translate status --task-id xyz789 --wait --format json
# run_in_background: true
```

**完成后**：

→ AskUserQuestion (N8a)

**用户选**: 下载

```bash
$ pdf-cli translate download --task-id xyz789 --output ./scan_zh.pdf
```

**agent 回复**：
> 已下载到 ./scan_zh.pdf

---

## 示例 3：未登录用户尝试高级翻译

**用户**: `高级翻译 ./paper.pdf`

**agent 思考**：用户给了路径 + 高级。

```bash
# N1
$ pdf-cli auth status   # exit 1 → 未登录
```

**agent 思考**：未登录 + 用户说"高级" → 直接走 N3b 的"是"分支，不必再问。

**agent 回复**（终止流程）：
> 高级翻译需要登录账号。请先运行：
> 
>     pdf-cli auth login --email you@example.com
> 
> 登录完成后再次发起翻译即可。

不调用任何后续命令，等用户登录后重新发起。

---

## 示例 4：普通用户走完整高级流程

**用户**: `翻译 ./report.pdf`

**agent 思考**：只有路径，所有节点都要走。

```bash
# N1
$ pdf-cli auth status   # exit 0 → 已登录
```

→ AskUserQuestion (N3a)

**用户选**: 高级翻译

```bash
# N2
$ pdf-cli user profile --format json
# → {"vipLevel": 0, ...}    普通用户

# N4 数据
$ pdf-cli translate languages --format json

# N5 数据
$ pdf-cli translate engines --format json
```

**agent**：先问语言 + 引擎类型 + OCR。在打开问题前打印：
> 你不是会员，可选普通引擎；高级引擎仅会员可用。
> 当前限额：每日 2 次 / 每次最多 600 字符。

→ AskUserQuestion (合并 N4-S1 / N5-S1 / N6)

**用户选**: zh-CN / 普通引擎 / 否

**agent**：根据用户选的引擎类型，展示普通组前 3 个引擎 + `other`。

→ AskUserQuestion (N5-S2)

**用户选**: other

**agent 回复**：
> 没有命中想要的引擎。请直接告诉我引擎名称，例如：
>
> - 用 chatgpt-4.1
> - 引擎 deepseek-v3.1
>
> 也可以先运行：
>
> `pdf-cli translate engines`
>
> 查看完整列表（含倍率），再告诉我 engineName。

**用户**: `google`

```bash
$ pdf-cli translate upload \
    --file ./report.pdf \
    --advanced \
    --to zh-CN \
    --engine google \
    --format json
# → {"taskId": "def456", ...}

# N7
$ pdf-cli translate status --task-id def456 --wait --format json
# run_in_background
```

**完成后** → N8a → 下载或稍后。

---

## 示例 5：arXiv 论文

**用户**: `翻译 arXiv 2301.00001 翻成中文`

**agent 思考**：arxiv 分支，跳过 N1/N2/upload，直接走 arxiv 命令。N4 已指定 (zh-CN)。

→ AskUserQuestion (N5 + N6 合并问)

**用户选**: google / 否

```bash
$ pdf-cli translate arxiv \
    --arxiv-id 2301.00001 \
    --to zh-CN \
    --engine google \
    --format json
# → {"taskId": "arxiv_xxx", ...}

# N7 N8 同上
```

---

## 反例：不要这样做

### ❌ 调 upload 不传 flag
```bash
$ pdf-cli translate upload --file ./paper.pdf
```
agent 不是 TTY，CLI 会回退默认值，跳过用户决策。**永远传齐 `--free`/`--advanced` + `--to`。**

### ❌ 替用户决定免费/高级
用户说 "翻译 X.pdf"（没说类型），agent 直接走免费路径。**应该用 AskUserQuestion 问 N3a/N3b。**

### ❌ 未登录走高级时自己调 auth login
```bash
$ pdf-cli auth login --email guess@example.com
```
登录需要邮箱+验证码交互，agent 无法完成。**只打印登录命令并停止。**

### ❌ 不解析 exit code，硬解析中文 stdout
```bash
$ pdf-cli auth status | grep "已登录"
```
脆弱，国际化时会坏。**用 `$?` exit code。**

### ❌ status 不用 run_in_background
翻译可能 5 分钟，前台 wait 会阻塞主对话。**`run_in_background: true`，让用户能聊别的。**
