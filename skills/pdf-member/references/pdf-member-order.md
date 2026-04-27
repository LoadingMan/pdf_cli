# member order

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查看订单列表和详情。

## 命令

### 订单列表

```bash
# 默认列表
pdf-cli member order list

# 指定分页
pdf-cli member order list --page 1 --page-size 10

# JSON 格式
pdf-cli member order list --format json
```

### 订单详情

```bash
pdf-cli member order get --order-no <订单号>
```

## 参数

### order list

| 参数 | 必填 | 说明 |
|------|------|------|
| `--page <int>` | 否 | 页码（默认 1） |
| `--page-size <int>` | 否 | 每页数量（默认 20） |
| `--format <type>` | 否 | 输出格式 |

### order get

| 参数 | 必填 | 说明 |
|------|------|------|
| `--order-no <string>` | 是 | 订单号 |
| `--format <type>` | 否 | 输出格式 |

## API

| 操作 | 方法 | 路径 |
|------|------|------|
| 列表 | POST JSON | `user/trade/list` |
| 详情 | GET | `user/trade/get?orderNo=<order-no>` |

- 需要登录
