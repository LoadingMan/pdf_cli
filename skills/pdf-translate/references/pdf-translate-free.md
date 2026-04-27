# translate free

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

免费翻译，无需登录即可使用（所有用户都可用，包括游客/普通/会员）。**目标语言缺省时弹出交互选择菜单**。

## 命令

```bash
# 交互选择语言（推荐）
pdf-cli translate free --file-key <key>

# 指定目标语言
pdf-cli translate free --file-key <key> --to zh
pdf-cli translate free --file-key <key> --to zh --from en --file-format pdf
pdf-cli translate free --file-key <key> --to zh --pages "1,2,3"
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--file-key` | 是 | 上传后获得的文件 key（`tmpFileName`） |
| `--to` | 否* | 目标语言代码（如 zh, en, ja）；缺省时交互选择 |
| `--from` | 否 | 源语言代码，不填则自动检测 |
| `--file-format` | 否 | 文件格式（如 pdf, docx） |
| `--pages` | 否 | 指定翻译页码，逗号分隔 |
| `--visitor-email` | 否 | 游客邮箱 |

\* 非 TTY 环境下缺省 `--to` 会报错（无法交互）

## 流程（对应流程图）

1. 打印**游客/免费用户身份提示**：每日次数、文件大小上限、页数上限、结果不可下载（游客时）
2. `--to` 缺省时调用 `core/pdf/lang/list` 拉取语言列表，交互选择
3. 预检 `freeTransLimitNumPerDay / freeTransYetUsedNum`，今日配额用尽则抛 `quota` 错误
4. `POST core/pdf/free/translate` 发起翻译
5. 打印 task-id、record-id、排队数量。游客完成后 `translate status --wait` 会自动在控制台打印译文（需 `pdftotext`）

## 输出示例

```
[游客模式] 未登录 — 将以游客身份走免费翻译流程。
  · 翻译结果仅可在线查看，不可通过 CLI 下载保存
  · 如需下载结果，请先执行 pdf-cli auth login --email you@example.com
  · 免费翻译今日次数: 0/3
  · 免费翻译文件大小上限: 10.0 MB
  · 免费翻译页数上限: 100 页

选择目标语言  (↑/↓ 选择, Enter 确认, Ctrl-C 取消)
  English  (en)
▶ 中文  (zh)
  日本語  (ja)
  ...

免费翻译已发起
  task-id    : 20260407171457_h44sx4ht-xhp77-zh
  record-id  : 21247
  排队数量   : 2
  译文预览 URL: https://res.doclingo.ai/pdf/20260407171457_h44sx4ht-xhp77-zh.pdf  (翻译完成后在浏览器中打开可查看/打印)

下一步: pdf-cli translate status --task-id 20260407171457_h44sx4ht-xhp77-zh --wait
```

## API

- **端点**: `POST core/pdf/free/translate`
- **认证**: 不需要
- **请求体**: `{fileKey, targetLang, sourceLang, fileFmtType, pages, visitorEmail}`

## 响应字段

| 字段 | 说明 |
|------|------|
| `blobFileName` | 翻译结果文件名，去掉扩展名即为 task-id |
| `id` | 操作记录 ID（record-id） |
| `existFreeTransCount` | 排队中的免费翻译数量 |

## Notes

- 无需登录，适合游客使用；登录用户也可主动选择免费流程节省配额
- 免费翻译有排队限制，需等待前面的任务完成
- 游客完成翻译后，CLI 会打印公开预览 URL；登录后可改用 `translate download` 保存文件
- 上传文件走 `translate upload`，游客也可上传（自动以 `freeTag=1` 走免费通道）
