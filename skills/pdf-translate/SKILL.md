---
name: pdf-translate
version: 1.3.0
description: "pdf-cli 翻译模块：PDF 文档翻译全流程。translate upload 按登录态自动分流：已登录问是否免费翻译；未登录问是否高级翻译（是则提示登录，否则默认免费流程）。交互选择目标语言（来自 translate languages）、引擎（translate engines，会员可选高级）、OCR。游客完成后在控制台打印所有译文（需 pdftotext）；登录用户可下载到本地。"
metadata:
  requires:
    bins: ["pdf-cli"]
  cliHelp: "pdf-cli translate --help"
---

# translate

**CRITICAL — 开始前 MUST 先用 Read 工具读取 [`../pdf-shared/SKILL.md`](../pdf-shared/SKILL.md)，其中包含认证、配置、错误处理**

## Core Concepts

- **file-key**: 上传文件后获得的唯一标识（`tmpFileName`），用于发起翻译
- **task-id**: 翻译任务标识（`blobFileName` 去掉扩展名），用于查询状态和下载结果
- **record-id**: 操作记录 ID（数字），用于下载翻译结果和查看记录详情

## 翻译流程图（对应 CLI 行为）

```
开始 (translate upload --file ./paper.pdf)
 ↓
是否已经登录？ （CLI 自动判断登录态）
 │
 ├─ 已登录 ──→ 是否使用免费翻译？
 │              ├─ 是 → 上传（freeTag=1）→ 选择目标语言 → 免费 CLI
 │              └─ 否 → 上传（freeTag=0）→ 用户是否为会员？
 │                       ├─ 会员：选择目标语言 + 引擎（普通/高级）
 │                       └─ 普通：选择目标语言 + 仅普通引擎（2次/天·600字/次）
 │                       ↓
 │                      是否启用 OCR？
 │                       ├─ 是：翻译 CLI 添加 OCR 标记
 │                       └─ 否：使用高级翻译 CLI
 │
 └─ 未登录 ──→ 是否使用高级翻译？
                ├─ 是 → 提示: pdf-cli auth login --email you@example.com  并退出
                └─ 否 → 上传（freeTag=1）→ 提示游客身份及功能限制
                        → 选择目标语言 → 免费 CLI
 ↓
查询翻译状态 (translate status --task-id xxx --wait)
 ↓
是否是登录用户？
 ├─ 是 → translate download 下载到本地
 └─ 否 → 游客在控制台打印所有译文（通过系统 pdftotext 从译文 PDF 提取）
 ↓
结束
```

## 交互式流程

`translate upload`、`translate free`、`translate start` 在 TTY 下弹出**箭头键选择菜单**（↑/↓ 或 j/k 移动，Enter 确认，Ctrl-C 取消）。
菜单直接读写 `/dev/tty`，不受 stdin/stdout 重定向影响；真正无控制终端时安全回退默认值。

| 决策节点 | 触发命令 | 行为 |
|---|---|---|
| 是否已经登录？ | `translate upload` | 系统自动判断，不问用户 |
| 是否使用免费翻译？ | `translate upload`（已登录时） | 交互选择是/否 |
| 是否使用高级翻译？ | `translate upload`（未登录时） | 交互选择是/否；是则提示登录 |
| 选择目标语言 | `translate free` / `translate start` | 从 `translate languages` API 拉取，交互选择 |
| 选择翻译引擎 | `translate start` | 从 `translate engines` API 拉取，非会员仅显示普通引擎 |
| 是否启用 OCR | `translate start` | 交互选择是/否 |

## 命令概览

| 命令 | 说明 | 需要登录 |
|------|------|----------|
| [`languages`](references/pdf-translate-languages.md) | 获取支持的语言列表 | 否 |
| [`engines`](references/pdf-translate-engines.md) | 获取可用翻译引擎列表 | 否 |
| [`upload`](references/pdf-translate-upload.md) | 完整入口：按登录态分流→上传→接力翻译 | 可选 |
| [`start`](references/pdf-translate-start.md) | 直接发起高级翻译（脚本场景）；交互选择语言/引擎/OCR | 是 |
| [`free`](references/pdf-translate-free.md) | 直接发起免费翻译（脚本场景）；交互选择语言 | 否 |
| [`arxiv`](references/pdf-translate-arxiv.md) | arXiv 论文下载并翻译 | 否 |
| [`arxiv-info`](references/pdf-translate-arxiv-info.md) | 查询 arXiv 论文摘要 | 否 |
| [`text`](references/pdf-translate-text.md) | 文本内容翻译 | 是 |
| [`status`](references/pdf-translate-status.md) | 查询翻译进度 | 否（游客完成时打印预览 URL） |
| [`continue`](references/pdf-translate-continue.md) | 继续暂停的翻译 | 是 |
| [`cancel`](references/pdf-translate-cancel.md) | 取消翻译任务 | 是 |
| [`download`](references/pdf-translate-download.md) | 下载翻译结果；游客仅打印预览 URL | 是（游客打印 URL） |
| [`history`](references/pdf-translate-history.md) | 查看翻译记录 | 是 |

## 快速开始

```bash
# 推荐：一条命令走完整流程
pdf-cli translate upload --file ./paper.pdf
# → (根据登录态) 交互选择：是否免费 / 是否高级
# → 上传
# → 交互选择：目标语言 [/ 引擎 / OCR]
# → 翻译已发起，打印 task-id
pdf-cli translate status --task-id <task-id> --wait
# → 登录用户：提示运行 download；游客：自动控制台打印译文

# 登录用户下载
pdf-cli translate download --task-id <task-id>

# 分步调用（脚本/CI 场景，仍可单独使用）
pdf-cli translate free  --file-key <key> --to zh
pdf-cli translate start --file-key <key> --to zh --engine google --ocr

# arXiv 论文翻译（不经过 upload）
pdf-cli translate arxiv --arxiv-id 2301.00001 --to zh --engine 1

# 文本内容翻译（SSE 流式返回）
pdf-cli translate text --text "Hello World" --to zh-CN
```

## 用户身份与能力矩阵

| 身份 | 上传 | 高级翻译 | 免费翻译 | 引擎选择 | OCR | 结果输出 |
|------|------|----------|----------|----------|-----|----------|
| 游客（未登录） | 仅免费 | ❌（引导登录） | ✅ | 不适用 | 不适用 | **控制台打印译文**（pdftotext 提取） |
| 普通用户（已登录） | ✅ | ✅（限 2次/天 · 600字/次） | ✅ | 仅普通引擎 | ✅ | 下载到本地 |
| 会员用户 | ✅ | ✅（无上述限制，消耗会员字符） | ✅ | 普通+高级 | ✅ | 下载到本地 |

## 用户限制条件预检（Homepage Config）

在执行以下命令前，CLI 会自动调用 `GET user/config/homepage` 获取当前用户（或游客）的限制条件，并按以下规则做本地预检：

| 命令 | 预检字段 | 拦截条件 |
|------|----------|----------|
| `upload`（未登录） | `freeTransMaxFileSize` | 文件大小超过后端上限 → `quota` 错误 |
| `free` | `freeTransLimitNumPerDay` / `freeTransYetUsedNum` | 今日免费次数已用尽 → `quota` 错误 |
| `start` | `remainTransCountByDay` / `remainCharsCountByDay` | 两者都为 0 → `quota` 错误 |
| `start --engine <premium>` | `highLevelFlag` + `vipLevel` | 非会员选择高级引擎 → `quota` 错误 |
| `arxiv` | 同 `start` | 同上 |

- 预检失败直接抛出 `quota` 错误并退出，避免无谓的请求与排队。
- Homepage 接口本身失败时，CLI 打印 stderr 警告但**不阻塞流程**，由后端业务错误码兜底。
- 接口已登录时返回用户额度（`remainCharsCountByDay` 等），未登录时返回游客维度（`freeTrans*`、`visitor*`）。

## 游客查看译文（控制台打印译文内容）

流程图规定游客不可下载到本地，但**在控制台打印所有译文内容**。
CLI 实现：下载译文 PDF 到临时文件 → 调用系统 `pdftotext` 提取文本 → 打印到 stdout → 清理临时文件。

触发时机：

- `translate status`（done 分支，未登录）：自动打印译文
- `translate download`（未登录）：不再拒绝，直接打印译文

降级策略：

- 系统无 `pdftotext`（`apt install poppler-utils` 可安装）时，改打印预览 URL `https://res.doclingo.ai/pdf/<task-id>.pdf`
- PDF 下载失败（任务未完成 / task-id 过期）：返回 `not_found` 错误
- `pdftotext` 执行失败：降级为打印预览 URL

## Important Notes

- 上传阶段的 `freeTag` 取决于**登录态**（未登录=1，已登录=0），与"高级/免费"选择解耦；因此"是否使用高级翻译"询问放到 upload 完成之后。
- 上传返回的 `file-key` 是 `tmpFileName` 字段（优先），不是 `sourceFileId`
- 发起翻译返回的 `task-id` 是 `blobFileName` 去掉扩展名，用于 status 查询
- 发起翻译返回的 `record-id` 是数字 ID，用于 download 和 records 查询
- 翻译 API 要求 `ocrFlag` 参数，CLI 通过 `--ocr` flag 或交互选择控制（默认关闭）
- status 查询的响应格式为 `{taskId: {status, translateRate, ...}}`，是嵌套结构
- 交互式菜单直接读写 `/dev/tty`，绕过 stdin/stdout 重定向导致的 `IsTerminal` 误判；无控制终端时安全回退到默认值
- `start` 命令支持 `--term-ids`（术语表）、`--prompt-type`（翻译风格）等高级参数
- `arxiv` 命令直接通过 arXiv ID 下载并翻译，无需先上传文件
