# translate arxiv-info

查询 arXiv 论文的摘要信息。

## 用法

```bash
pdf-cli translate arxiv-info --arxiv-id <id>
pdf-cli translate arxiv-info --arxiv-id 2301.00001
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--arxiv-id` | 是 | arXiv 论文 ID（如 2301.00001） |

## API

- **端点**: `GET core/pdf/query/arxiv/summary`
- **认证**: 不需要

## 响应字段

| 字段 | 说明 |
|------|------|
| `title` | 论文标题 |
| `authors` | 作者列表 |
| `summary` | 论文摘要 |
