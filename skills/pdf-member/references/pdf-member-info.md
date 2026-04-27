# member info

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看会员配置信息。

## 命令

```bash
pdf-cli member info

# JSON 格式
pdf-cli member info --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--format <type>` | 否 | 输出格式 |

## API

- GET `user/config/vip/cfg`
- 需要登录

## Notes

- 当前版本该接口可能返回 400 错误，属于服务端已知问题
