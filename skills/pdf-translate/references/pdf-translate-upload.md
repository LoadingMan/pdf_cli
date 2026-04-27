# translate upload

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

PDF 翻译总入口。按流程图：**先根据登录态自动分流**，再弹出身份对应的选择菜单，然后上传并接力翻译请求。

## 命令

```bash
pdf-cli translate upload --file ./paper.pdf
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--file <path>` | 是 | 待翻译 PDF 路径 |
| `--format <type>` | 否 | 输出格式 |

## 流程（对应新版流程图）

### 已登录用户

1. 弹出菜单：**是否使用免费翻译？**
2. 选"是" → `freeTag=1` 上传 → 走免费流程（仅需选择目标语言）
3. 选"否" → `freeTag=0` 上传 → 走高级流程（选择目标语言 → 引擎 → OCR）

### 未登录用户（游客）

1. 弹出菜单：**是否使用高级翻译？**
2. 选"是" → 返回 `AuthError`：提示 `pdf-cli auth login --email you@example.com`，要求登录后重跑
3. 选"否" → 打印游客身份与免费限制 → `freeTag=1` 上传 → 走免费流程（选择目标语言）

## 上传三步（`doUpload`）

1. `POST core/pdf/trans/aws/pre/upload` → 拿到 `awsUploadUrl`、`blobFileName`
2. `PUT awsUploadUrl`（multipart → S3 直传）
3. `POST core/pdf/file/new/upload` → 注册文件，拿到 `tmpFileName` 作为 file-key

## 输出示例（游客）

```
是否使用高级翻译？（需要登录；否则走默认免费流程）  (↑/↓ 选择, Enter 确认, Ctrl-C 取消)
  是
▶ 否

[游客模式] 未登录 — 将以游客身份走免费翻译流程。
  · 翻译结果仅可在控制台打印，不可下载保存
  · 如需保存为本地文件，请先执行 pdf-cli auth login --email you@example.com
  · 免费翻译今日次数: 0/3
  · 免费翻译文件大小上限: 10.0 MB
正在准备上传...
正在上传文件...
正在注册文件...
上传成功
  file-key : 20260424180007_xxxx

选择目标语言  (↑/↓ 选择, Enter 确认, Ctrl-C 取消)
▶ English  (en)
  中文  (zh)
  ...

免费翻译已发起
  task-id    : 20260424180007_xxxx-yyyy-zz-en
```

## Notes

- 交互菜单直接读写 `/dev/tty`，不会被 IDE 终端 / tmux / pipe 重定向阻断
- 非 TTY 环境（CI/脚本）下菜单返回默认值；推荐在脚本里改用 `translate free` / `translate start` 分步命令
- 上传完成后直接调用 `runFreeTranslate` 或 `runAdvancedTranslate`，因此 upload 成功即表示翻译已发起
- 游客结果后续通过 `translate status --wait` 完成时自动控制台打印译文（需 `pdftotext`）
