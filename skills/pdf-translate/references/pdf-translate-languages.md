# translate languages

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

获取支持的翻译语言列表。

## 命令

```bash
# 表格格式（默认）
pdf-cli translate languages

# JSON 格式
pdf-cli translate languages --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--format <type>` | 否 | 输出格式：`pretty`（默认）、`json` |

## 输出示例

```
  语言代码     语言名称
  --------  ----------
  zh-CN     简体中文
  en        英语
  ja        日语
  ko        韩语
  fr        法语
  ...
```

## API

- GET `core/pdf/lang/list`
- 响应中语言对象字段为 `code` 和 `name`
- 不需要登录
