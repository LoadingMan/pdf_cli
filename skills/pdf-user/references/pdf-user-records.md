# user records

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看用户操作记录（翻译、转换等）。

## 命令

### 列表

```bash
# 默认列表
pdf-cli user records list

# 指定分页
pdf-cli user records list --page-size 10

# JSON 格式
pdf-cli user records list --format json
```

### 详情

```bash
pdf-cli user records get --id <record-id>

# JSON 格式
pdf-cli user records get --id <record-id> --format json
```

## 参数

### records list

| 参数 | 必填 | 说明 |
|------|------|------|
| `--page-size <int>` | 否 | 每页数量（默认 20） |
| `--format <type>` | 否 | 输出格式 |

### records get

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id <string>` | 是 | 操作记录 ID |
| `--format <type>` | 否 | 输出格式 |

## 输出示例（list）

```
  ID     文件名        状态      时间
  -----  ---------  ------  -----------------------------
  21249  test.pdf   英语      2026-04-07T10:11:17.000+00:00
```

## API

| 操作 | 方法 | 路径 |
|------|------|------|
| 列表 | POST JSON | `user/operate/record/list/page` |
| 详情 | GET | `user/operate/record/get?operateRecordId=<id>` |

- 列表响应中 `dataList` 在顶层
- 需要登录
