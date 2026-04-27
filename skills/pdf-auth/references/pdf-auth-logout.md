# auth logout

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

退出当前登录状态，清除本地 token。

## 命令

```bash
pdf-cli auth logout
```

## 参数

无。

## 行为说明

1. 如果当前已登录，先通知服务端登出（GET `user/basic/login/out`），忽略服务端错误
2. 清除本地配置中的 token
3. deviceId 保留不清除

## 输出示例

```
OK: 已退出登录
```
