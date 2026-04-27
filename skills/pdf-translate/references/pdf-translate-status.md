# translate status

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

查询翻译任务的当前状态和进度。**按新流程图：完成时登录用户提示下载命令；游客自动在控制台打印所有译文**。

## 命令

```bash
# 查询一次
pdf-cli translate status --task-id <task-id>

# 轮询等待完成（每 5 秒查询一次，完成时自动打印译文或下一步）
pdf-cli translate status --task-id <task-id> --wait
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--task-id <string>` | 是 | 翻译任务 ID（来自 upload 接力 / start / free 的 task-id） |
| `--wait` | 否 | 持续轮询直到翻译完成或失败 |
| `--format <type>` | 否 | 输出格式 |

## 输出示例

### 登录用户 — 完成

```
  状态   : done
  进度   : 100%

下一步: pdf-cli translate download --task-id <task-id>
```

### 游客 — 完成（自动打印译文）

```
  状态   : done
  进度   : 100%

翻译已完成（游客模式 — 在控制台打印译文内容）：
------------------------------------------------------------
<译文全文，经系统 pdftotext 提取>
...
------------------------------------------------------------
（如需保存为本地文件，请先执行 pdf-cli auth login --email you@example.com）
```

### 翻译中

```
  状态   : translating
  进度   : 45%
  等待中...
```

### 失败

```
  状态   : fail
  失败原因: <reason>
```

## API

- GET `core/pdf/query/status?queryFileKey=<task-id>`
- 响应格式为嵌套结构：`{taskId: {status, translateRate, failReason}}`
- 状态值：`done`/`success`/`finish`/`finished`（视为完成）、`fail`、`cancel`、`translating` 等
- 本接口公开，不需要登录

## 依赖

游客在 `done` 分支会调用系统 `pdftotext`（来自 `poppler-utils`）提取译文。缺失时降级为打印预览 URL。

## Notes

- `--wait` 下轮询间隔 5 秒，直到进入终态（done/fail/cancel）
- 失败时退出码非 0，类型 `task_failed`，附带 `task_id` 与 `fail_reason` 详情
- 游客自动打印译文的行为与 `translate download` 游客分支一致
