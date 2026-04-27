# user api-key

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

管理 API Key，用于平台接口认证。

## 命令

### 列表

```bash
pdf-cli user api-key list
```

### 创建

```bash
pdf-cli user api-key create --name "my-key"
```

### 删除

```bash
pdf-cli user api-key delete --id <key-id>
```

## 参数

### api-key create

| 参数 | 必填 | 说明 |
|------|------|------|
| `--name <string>` | 是 | API Key 名称 |

### api-key delete

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id <string>` | 是 | API Key ID |

## API

| 操作 | 方法 | 路径 |
|------|------|------|
| 列表 | GET | `user/apicommon/sk/list` |
| 创建 | POST JSON | `user/apicommon/sk/add` |
| 删除 | POST JSON | `user/apicommon/sk/del` |

- 需要登录
