# auth login

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

使用邮箱和密码登录账号，获取 token 并保存到本地配置。

## 命令

```bash
# 登录（交互式输入密码）
pdf-cli auth login --email you@example.com

# 指定输出格式
pdf-cli auth login --email you@example.com --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--email <string>` | 是 | 登录邮箱地址 |

## 行为说明

1. 执行后会提示 `请输入密码:`，通过终端安全输入（不回显）
2. 发送 JSON POST 请求到 `user/basic/base/login`
3. 请求自动携带 `deviceId` 和 `clientType: cli` header
4. 登录成功后 token 自动保存到 `~/.config/pdf-cli/config.json`

## 输出示例

```
请输入密码:
OK: 登录成功
```

## 错误场景

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| `请提供邮箱地址` | 未传 --email | 添加 --email 参数 |
| `密码不能为空` | 未输入密码 | 输入密码后回车 |
| `HTTP 400` | 邮箱或密码错误 | 检查邮箱和密码 |
