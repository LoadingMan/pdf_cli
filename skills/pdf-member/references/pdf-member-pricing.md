# member pricing

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看会员价格配置。

## 命令

```bash
pdf-cli member pricing

# JSON 格式
pdf-cli member pricing --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--format <type>` | 否 | 输出格式 |

## API

- GET `user/config/all/price/cfg`
- 需要登录
- 返回各会员等级的价格配置
