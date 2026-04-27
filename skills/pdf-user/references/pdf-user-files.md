# user files

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看用户上传的文件列表。

## 命令

```bash
# 默认列表
pdf-cli user files list

# 指定分页
pdf-cli user files list --page-size 10

# JSON 格式
pdf-cli user files list --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--page-size <int>` | 否 | 每页数量（默认 20） |
| `--format <type>` | 否 | 输出格式 |

## 输出示例

```
  ID     文件名                           大小        上传时间
  -----  ----------------------------  --------  -----------------------------
  32213  test.pdf                      0.000568  2026-04-07T10:11:16.000+00:00
  32212  test.pdf                      0.000302  2026-04-07T09:35:50.000+00:00
```

## API

- POST JSON `user/source/file/list`
- 请求体：`{pageSize: <size>}`
- 响应中 `dataList` 在顶层
- 文件名字段可能为 `origFileName`、`originFileName` 或 `fileName`
- 需要登录
