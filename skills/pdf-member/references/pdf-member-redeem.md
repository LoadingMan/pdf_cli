# member redeem

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

使用兑换码兑换会员。

## 命令

```bash
pdf-cli member redeem --code <兑换码>
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--code <string>` | 是 | 兑换码 |

## API

- POST JSON `user/vipcode/bind`
- 请求体：`{code: <code>}`
- 需要登录
