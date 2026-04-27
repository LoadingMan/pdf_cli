# translate cancel

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

取消一个进行中的翻译任务。

## 命令

```bash
pdf-cli translate cancel --task-id <task-id>
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--task-id <string>` | 是 | 翻译任务 ID |

## API

- POST JSON `core/pdf/cancel/trans`
- 请求体：`{operateRecordId: <task-id>}`
- 需要登录

## Notes

- 该接口在当前后端版本中可能未启用，调用可能返回错误
