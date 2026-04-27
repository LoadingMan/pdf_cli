# tools job

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

管理异步任务状态与结果下载。

## 命令

### 查询任务状态

```bash
pdf-cli tools job status --query-key <key>

# 兼容旧参数
pdf-cli tools job status --job-id <key>
```

### 下载任务结果

```bash
pdf-cli tools job download --query-key <key>

# 指定输出路径
pdf-cli tools job download --query-key <key> --output result.pdf
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--query-key <string>` | 是 | 任务查询键 |
| `--job-id <string>` | 否 | 旧参数别名，内部等同于 `queryKey` |
| `--output, -o <path>` | 否 | 输出文件路径，仅 download 使用 |

## 输出

`status` 的 JSON 输出包含 `queryKey`、`state`、`result` 等字段。

## 接口映射

- 状态查询：`GET core/tools/operate/status?queryKey=...`
- 结果下载：根据任务结果中的文件名下载

## Notes

- 主文档术语是 `queryKey`。
- 旧参数别名仅用于兼容。
- 大多数 tools 命令会自动轮询并下载，无需手动调用。
