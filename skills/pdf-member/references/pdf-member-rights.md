# member rights

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看各等级会员权益配置。

## 命令

```bash
pdf-cli member rights

# JSON 格式
pdf-cli member rights --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--format <type>` | 否 | 输出格式 |

## API

- GET `user/basic/get/vip/functions`
- 需要登录
- 返回各会员等级对应的功能权益列表
