# translate arxiv

通过 arXiv ID 下载论文并翻译。

## 用法

```bash
pdf-cli translate arxiv --arxiv-id <id> --to <语言代码>
pdf-cli translate arxiv --arxiv-id 2301.00001 --to zh --engine 1
pdf-cli translate arxiv --arxiv-id 2301.00001 --to zh --engine 1 --term-ids "1,2"
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--arxiv-id` | 是 | arXiv 论文 ID（如 2301.00001） |
| `--to` | 是 | 目标语言代码（如 zh, en, ja） |
| `--from` | 否 | 源语言代码，不填则自动检测 |
| `--engine` | 否 | 翻译引擎 ID |
| `--ocr` | 否 | 启用 OCR 模式 |
| `--term-ids` | 否 | 术语表 ID，多个用逗号分隔 |

## API

- **端点**: `POST core/pdf/arxiv/translate`
- **认证**: 不需要（游客可用）

## 响应字段

| 字段 | 说明 |
|------|------|
| `blobFileName` | 翻译结果文件名，去掉扩展名即为 task-id |
| `id` | 操作记录 ID（record-id） |
| `origFileName` | 原始文件名 |

## 相关命令

- `pdf-cli translate arxiv-info --arxiv-id <id>` — 查询论文摘要信息

## 注意

- 无需先上传文件，直接通过 arXiv ID 下载并翻译
- 支持选择翻译引擎和术语表
