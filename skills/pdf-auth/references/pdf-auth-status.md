# auth status

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看当前登录状态和用户信息。

## 命令

```bash
# 查看登录状态（pretty 格式）
pdf-cli auth status

# JSON 格式输出
pdf-cli auth status --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--format <type>` | 否 | 输出格式：`pretty`（默认）、`json` |

## 输出示例

### 已登录

```
登录状态: 已登录
  昵称           : doclingo_user
  会员等级         : 1
  剩余额度         : 5000
```

### 未登录

```
当前未登录
Hint: 请执行 pdf-cli auth login --email you@example.com
```

### JSON 格式

```json
{
  "userEmail": "user@example.com",
  "nickName": "doclingo_user",
  "vipLevel": 1,
  "remainScore": 5000
}
```

## API

- GET `user/basic/token/userinfo`
- 需要登录（携带 token header）
