# user update

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

修改当前用户的个人信息。

## 命令

```bash
pdf-cli user update --name "新昵称"
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--name <string>` | 是 | 新昵称 |

## API

- POST JSON `user/basic/modify/userinfo`
- 请求体：`{nickName: <name>}`
- 需要登录
