# translate engines

获取可用的翻译引擎列表。

## 用法

```bash
pdf-cli translate engines
pdf-cli translate engines --format json
```

## 参数

无参数。

## API

- **端点**: `GET core/pdf/engines`
- **认证**: 不需要

## 响应

返回引擎对象映射 `{<engineKey>: {...}}`，每个引擎包含：

| 字段 | 类型 | 含义 |
|------|------|------|
| `engineId` | int | 数字 ID |
| `engineName` | string | 引擎代码（传给 `--engine` 的值，如 `google` / `chatgpt-4.1`）|
| `engineShowName` | string | 用户可见的展示名 |
| `highLevelFlag` | 0/1 | `1` = 高级引擎（仅会员可用）；`0` = 普通引擎（所有登录用户可用）|
| `showFlag` | 0/1 | `0` = 对外不开放，UI 应隐藏 |
| `tokenCostRatio` | string | 字符消耗倍率，如 `"1"` / `"4"`；翻译 1 字符按倍率扣减会员字符额度 |
| `userDefaultFlag` | 0/1 | 普通用户默认引擎 |
| `vipDefaultFlag` | 0/1 | 会员默认引擎 |
| `iconImgUrl` / `iconRgbValue` | string | UI 图标信息 |

## 示例输出

`pretty` 格式（默认）按 `普通引擎 / 高级引擎` 分组打印：

```
普通引擎
  引擎名称        代码           倍率
  Google         google         1
  DeepL          deepl          1

高级引擎（仅会员可用）
  引擎名称           代码              倍率
  chatgpt-4.1-mini   chatgpt-4.1-mini  1
  chatgpt-4.1        chatgpt-4.1       4
  Claude Sonnet      claude-sonnet     6
```

`--format json` 返回原始映射，便于脚本/Agent 处理。

## 注意

- `--engine` flag 接受 `engineId`（数字）或 `engineName`（字符串）；推荐用 `engineName` 更可读
- `showFlag != 1` 的引擎不应展示给用户
- 普通用户选择 `highLevelFlag == 1` 的引擎会被后端拒绝（`quota` 错误）
- `tokenCostRatio` 仅在会员高级翻译时影响字符额度扣减；免费翻译不计费
