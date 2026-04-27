---
name: pdf-translate-flow
version: 1.0.0
description: "pdf-cli 翻译流程的 agent 主导编排。当用户要求翻译 PDF 且希望由 Claude 在每个关键决策节点暂停询问（免费/高级、目标语言、引擎、OCR）时使用。在每个决策点用 AskUserQuestion 工具问用户，然后用带显式参数的 CLI 子命令调用，绕过 CLI 自带的 TTY 菜单。用户已经在自然语言里表达过的决策（如『翻成日文』『高级翻译』）跳过对应节点。"
metadata:
  requires:
    bins: ["pdf-cli"]
    tools: ["AskUserQuestion", "Bash"]
---

# pdf-translate-flow

**CRITICAL — 开始前 MUST**：
1. Read [`../pdf-shared/SKILL.md`](../pdf-shared/SKILL.md) 了解认证、配置、错误处理
2. Read [`../pdf-translate/SKILL.md`](../pdf-translate/SKILL.md) 了解 CLI 子命令细节
3. 本 skill 假定 `AskUserQuestion` 工具可用（deferred tool，需要时通过 ToolSearch 加载 `select:AskUserQuestion`）

## 这个 skill 做什么

把 **副本翻译流程图** 翻译成 agent 可执行的编排：
- **agent 主导决策**：每到关键节点用 `AskUserQuestion` 暂停问用户，不依赖 CLI 内部 TTY 菜单
- **显式参数调用**：调用 `translate upload` 时传齐 `--free/--advanced --to --engine --ocr` 等 flag，跳过命令内部交互
- **跳过已决策节点**：用户在初始请求里说过的（"翻成日文"/"高级翻译"/"开 OCR"）不再问

## 触发场景

用户消息匹配下列任一形态：
- "翻译 X.pdf" / "把 X.pdf 翻译成 Y" / "高级 / 免费翻译 X.pdf"
- "翻译 arXiv 2301.00001"（走 [arxiv 分支](references/nodes.md#arxiv-分支)）
- 接续动作："继续之前的翻译" / "下载译文 task-id=…"（跳到节点 7/8）

不触发：
- 纯文本翻译（用 `pdf-cli translate text` 即可，无需流程编排）
- 用户明确说"用 CLI 交互菜单"

## 流程图（对应原图节点）

```
开始
 ↓
[N1] 检查登录态 (CLI 自动)
 │
 ├─ 已登录 ──→ [N3a] AskUserQuestion: 是否使用免费翻译？
 │              ├─ 是 → [N4] 选语言 → upload --free --to <lang>
 │              └─ 否 → [N2] 检查会员态 (CLI 自动)
 │                       ├─ 会员 → [N4] 选语言 → [N5] 选引擎(普通/高级)
 │                       └─ 普通 → [N4] 选语言 → [N5] 仅普通引擎
 │                                ↓
 │                               [N6] AskUserQuestion: 是否启用 OCR？
 │                                ↓
 │                               upload --advanced --to <lang> --engine <e> [--ocr]
 │
 └─ 未登录 ──→ [N3b] AskUserQuestion: 是否使用高级翻译？
                ├─ 是 → 提示运行 `pdf-cli auth login --email …` 并停止
                └─ 否 → [N4] 选语言 → 提示游客限制 → upload --free --to <lang>
 ↓
[N7] translate status --task-id <id> --wait
 ↓
[N8] 输出译文
 ├─ 已登录 → AskUserQuestion: 是否下载到本地？
 │            ├─ 是 → translate download --task-id <id>
 │            └─ 否 → 仅打印 task-id 提示后续可下载
 └─ 未登录 → CLI 自动控制台打印译文（pdftotext 提取）
 ↓
结束
```

## 决策节点速查

| # | 节点 | 触发 | 数据来源 | 工具 |
|---|------|------|----------|------|
| N1 | 检查登录态 | 总是 | `pdf-cli auth status`（exit code）| Bash |
| N2 | 检查会员态 | 已登录且 N3a=否 | `pdf-cli user profile --format json`（`vipLevel`，**仅用于 N5-S1 前的 stdout 提示文案，不影响是否问**）| Bash |
| N3a | 已登录-免费 or 高级 | N1=已登录 | — | AskUserQuestion |
| N3b | 未登录-是否高级 | N1=未登录 | — | AskUserQuestion |
| N4 | 选目标语言 | 总是（除非用户已说）| `pdf-cli translate languages --format json` | AskUserQuestion 两阶段（S1: 3 语言 + "更多语言…" → S2: 3 语言 + "手动输入语言代码" → 文本输入）|
| N5 | 选引擎 | 走高级路径 | `pdf-cli translate engines --format json` | AskUserQuestion 两阶段（S1: 普通/高级 → S2: 3 引擎含倍率 + "手动输入引擎名称" → 文本输入）|
| N6 | 是否 OCR | 走高级路径 | — | AskUserQuestion |

> AskUserQuestion 工具每问最多 4 个选项（schema 强制）。**不依赖工具自动 Other**——实测它在 UI 里不一定可见，所以把"更多/手动输入"作为显式选项放进 options，占用 1 个槽位，确保用户一定看到下一步入口。用户进入手动输入路径后改用对话自由文本回复，agent 拉完整列表做精确/模糊匹配，未命中则提示用 `pdf-cli translate languages|engines` 查询并终止。
| N7 | 等待状态 | 翻译已发起 | `pdf-cli translate status --task-id … --wait` | Bash |
| N8 | 输出译文 | N7=done | 已登录: `download` / 未登录: 自动打印 | Bash |

## 编排步骤（伪代码）

```
# 用户: "翻译 ./paper.pdf"

# N1
$ pdf-cli auth status
→ logged_in? = (exit code == 0)

# 跳过条件预扫
already_decided = parse(用户原话)
  # 例如 "高级翻译 X.pdf 翻成日文" → free=false, lang="ja"

if logged_in:
    # N3a
    if "free" not in already_decided:
        free = AskUserQuestion("使用免费翻译还是高级翻译？", options=["免费", "高级"])
    if free:
        # N4
        if "lang" not in already_decided:
            langs = $ pdf-cli translate languages --format json
            lang = AskUserQuestion("选择目标语言", options=top_languages(langs))
        $ pdf-cli translate upload --file <path> --free --to <lang> --format json
    else:
        # N2
        vip = $ pdf-cli user profile --format json | jq .vipLevel
        # N4
        lang = ...
        # N5
        engines = $ pdf-cli translate engines --format json
        if vip == 0:
            engines = [e for e in engines if e.level != "premium"]
        engine = AskUserQuestion("选择翻译引擎", options=engines)
        # N6
        ocr = AskUserQuestion("是否启用 OCR（扫描件需要）？", options=["是", "否"])
        $ pdf-cli translate upload --file <path> --advanced --to <lang> --engine <engine> [--ocr] --format json
else:
    # N3b
    advanced = AskUserQuestion("使用高级翻译（需登录）还是免费翻译（游客可用）？", options=["高级（需登录）", "免费"])
    if advanced:
        echo "请先登录: pdf-cli auth login --email you@example.com"
        return
    # N4
    lang = ...
    echo "提示：游客模式下译文只在控制台打印，不能下载到本地。"
    $ pdf-cli translate upload --file <path> --free --to <lang> --format json

task_id = parse(upload_output).taskId

# N7
$ pdf-cli translate status --task-id <task_id> --wait

# N8
if logged_in:
    download = AskUserQuestion("翻译完成，是否下载到本地？", options=["下载", "稍后"])
    if download:
        $ pdf-cli translate download --task-id <task_id>
# 未登录: status --wait 在 done 时自动打印译文，无需额外步骤
```

## 详细说明

- 每个节点的触发条件、措辞模板、错误分支：[references/nodes.md](references/nodes.md)
- AskUserQuestion 调用模板（中文 question/options）：[references/ask-patterns.md](references/ask-patterns.md)
- 三个完整端到端示例：[references/examples.md](references/examples.md)

## 关键约束

- **不要替用户决策**：除非用户原话已表达，每个决策点必须用 `AskUserQuestion`
- **传齐 flag 跳交互**：调 `translate upload` 时必须同时给 `--free` 或 `--advanced`，加上 `--to`，避免落到 TTY 菜单
- **额度预检失败**：`upload` 自带 homepage 预检，遇 `quota` 错误直接报告给用户，不要重试
- **未登录走高级=终止**：N3b 选"高级"时只提示登录命令，**不要**直接调 `auth login`（需要邮箱+验证码交互）
- **错误码处理**：参考 [pdf-shared](../pdf-shared/SKILL.md)（`auth` / `quota` / `not_found` / `network`）
- **arxiv 分支特殊**：用户给 arXiv ID 时不走 upload，直接调 `translate arxiv --arxiv-id …`，但 N4/N5/N6 仍然询问

## Important Notes

- N3a/N3b 的"是否高级"决定 `freeTag`，与"会员/普通"是两个独立维度
- N5 引擎过滤：拿到 engines 列表后，未拿到 vipLevel>0 的用户**手动过滤** `level=premium` 的项再展示
- N4 语言选项过多时（>50 个），AskUserQuestion 只展示常用 top 8（zh/en/ja/ko/fr/de/es/ru），并加一个"其他（手动输入语言代码）"
- 用户原话已包含决策时（如 "翻成日文" → ja，"高级翻译" → advanced=true，"开 OCR" → ocr=true），**直接跳过对应节点**，不要重复问
- 任意节点用户说"取消" → 立即停止流程，不残留临时状态（CLI 上传后的 file-key 由后端 TTL 自动清理）
