# translate start

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

发起高级翻译任务（**需要登录**）。**目标语言、引擎、OCR 缺省时弹出交互选择菜单**。

## 命令

```bash
# 全交互（推荐）— 依次选择语言、引擎、OCR
pdf-cli translate start --file-key <key>

# 指定部分参数，其余交互选择
pdf-cli translate start --file-key <key> --to zh

# 全显式参数（适合脚本/CI）
pdf-cli translate start --file-key <key> --to zh --engine google --ocr

# 术语表 + 翻译风格
pdf-cli translate start --file-key <key> --to zh --engine 1 --term-ids "1,2" --prompt-type 1

# 指定文件格式
pdf-cli translate start --file-key <key> --to zh --file-format docx
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--file-key <string>` | 是 | 上传后获得的文件 key（来自 upload 命令） |
| `--to <string>` | 否* | 目标语言代码（如 `zh`、`en`、`ja`）；缺省时交互选择 |
| `--from <string>` | 否 | 源语言代码（可选，默认自动检测） |
| `--engine <string>` | 否 | 翻译引擎 id/name；缺省时交互选择（会员可选高级） |
| `--ocr` | 否 | 启用 OCR 模式（扫描件翻译）；未显式传入时交互询问 |
| `--term-ids <string>` | 否 | 术语表 ID，多个用逗号分隔 |
| `--file-format <string>` | 否 | 文件格式（如 pdf, docx） |
| `--prompt-type <int>` | 否 | 翻译风格/提示类型（0 表示不传） |
| `--format <type>` | 否 | 输出格式 |

\* 非 TTY 环境下缺省 `--to` 会报错（无法交互）

## 流程（对应流程图）

1. **登录校验** → 未登录返回 `AuthError`，提示 `auth login` 或改用 `translate free`
2. 获取 `vipLevel` 与 `homepage` 配置；打印会员/普通用户身份提示及今日剩余次数/字符
3. **选择目标语言**（缺省时）→ 从 `core/pdf/lang/list` 拉取后交互选择
4. **选择翻译引擎**（缺省时）→ 从 `core/pdf/engines` 拉取，按 `vipLevel` 过滤：
   - 会员：显示所有 `showFlag=1` 的引擎（普通+高级）
   - 普通用户：仅显示 `highLevelFlag=0` 的普通引擎
   - 首项 "使用默认引擎（服务端自选）" 可选
5. **是否启用 OCR**（未显式传时）→ 交互选择是/否
6. `translatePrecheck(kind="start")` 校验：
   - `remainTransCountByDay` 和 `remainCharsCountByDay` 都为 0 → `quota` 错误
   - 非会员选择高级引擎 → `quota` 错误
7. `POST core/pdf/translate` 发起任务，返回 `blobFileName` 与 `id`

## 输出示例

```
[高级翻译] 会员用户 — 可选用任意翻译引擎（普通/高级）。
  · 今日剩余次数: 48    今日剩余字符: 195000

选择目标语言  (↑/↓ 选择, Enter 确认, Ctrl-C 取消)
▶ 中文  (zh)
  English  (en)
  ...

选择翻译引擎  (↑/↓ 选择, Enter 确认, Ctrl-C 取消)
▶ 使用默认引擎（服务端自选）
  Google  [普通]  (id=1)
  DeepL  [高级]  (id=2)
  ...

是否启用 OCR 功能？（扫描件/图片型 PDF 建议启用）  (↑/↓ 选择, Enter 确认, Ctrl-C 取消)
  是
▶ 否

翻译已发起
  task-id    : 20260407171457_h44sx4ht-xhp77-auto-zh
  record-id  : 21247

下一步: pdf-cli translate status --task-id 20260407171457_h44sx4ht-xhp77-auto-zh
```

## API

- `POST core/pdf/translate`
- 请求体: `{fileKey, targetLang, ocrFlag, [sourceLang], [transEngineType], [termIds], [fileFmtType], [promptType]}`
- 响应 data 包含 `blobFileName`（去掉扩展名作为 task-id）和 `id`（record-id）
- 需要登录

## Notes

- `ocrFlag` 是必填 API 参数，CLI 默认 0；`--ocr` 或交互选"是"时为 1
- task-id 用于 `translate status` 查询进度；record-id 用于 `translate download` 下载结果
- `--term-ids` 支持多个术语表，逗号分隔
- `--prompt-type` 只在非 0 时传递给 API
- 非会员的高级翻译受后端配额约束（约 2次/天，600字/次），额度用尽会被 `translatePrecheck` 或后端业务码拦截
