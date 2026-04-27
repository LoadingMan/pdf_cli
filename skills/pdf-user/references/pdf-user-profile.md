# user profile

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看当前登录用户的个人信息。

## 命令

```bash
# pretty 格式
pdf-cli user profile

# JSON 格式
pdf-cli user profile --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--format <type>` | 否 | 输出格式 |

## 输出示例

```
  昵称           : doclingo_user
  邮箱           : user@example.com
  会员等级         : 1
  注册时间         : 2026-01-01T00:00:00.000+00:00
```

## API

- GET `user/basic/token/userinfo`
- 需要登录
