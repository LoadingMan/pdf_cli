# translate history

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看历史翻译记录列表。

## 命令

```bash
# 默认第 1 页，每页 20 条
pdf-cli translate history

# 指定分页
pdf-cli translate history --page 2 --page-size 10

# JSON 格式
pdf-cli translate history --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--page <int>` | 否 | 页码（默认 1） |
| `--page-size <int>` | 否 | 每页数量（默认 20） |
| `--format <type>` | 否 | 输出格式 |

## 输出示例

```
  ID     文件名        状态      时间
  -----  ---------  ------  -----------------------------
  21249  test.pdf   英语      2026-04-07T10:11:17.000+00:00
  21248  test.pdf   英语      2026-04-07T09:35:51.000+00:00
  21247  test.pdf   英语      2026-04-07T09:25:29.000+00:00
```

## API

- POST JSON `user/operate/record/list/page`
- 请求体：`{pageNo, pageSize}`
- 响应中 `dataList` 在顶层（不在 `data` 内）
- 列表项字段：`id`、`origFileName`、`operateTag`、`createTime`
- 需要登录
