# other version

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看版本历史记录。

## 命令

```bash
pdf-cli other version

# JSON 格式
pdf-cli other version --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--format <type>` | 否 | 输出格式 |

## API

- GET `user/history/version/list`
- 不需要登录

## Notes

- 当前版本该接口可能返回 500 错误，属于服务端已知问题
