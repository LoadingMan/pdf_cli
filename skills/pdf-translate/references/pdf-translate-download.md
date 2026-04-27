# translate download

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

下载翻译结果。按新流程图：**登录用户保存到本地文件；游客在控制台打印所有译文**（下载译文 PDF → 经 `pdftotext` 提取文本 → 打印到 stdout）。

## 命令

```bash
# 登录用户：下载到本地
pdf-cli translate download --task-id <task-id>
pdf-cli translate download --task-id <task-id> --output ./result.pdf

# 游客：控制台打印译文（自动触发，无需 --output）
pdf-cli translate download --task-id <task-id>
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--task-id <string>` | 是 | 支持 task-id（`blobFileName` 去扩展）或数字 record-id |
| `--output, -o <path>` | 否 | 登录用户输出路径（默认 `translated_<原文件名>`）；游客下忽略 |
| `--format <type>` | 否 | 输出格式 |

## 流程（对应新版流程图）

### 登录用户分支

1. 通过 `record-id` 调用 `user/operate/record/down/info` 获取 `blobFileName` 与 `originFileName`
2. 从下载源（主下载 URL → gdpdf → doclingo）依次拉取 PDF
3. 保存到 `--output` 指定路径或 `translated_<原文件名>`

### 游客分支（新）

1. 下载译文 PDF 到临时文件（`os.CreateTemp("", "pdf-cli-trans-*.pdf")`）
2. 调用系统 `pdftotext -layout -q <tmp> -`，stdout 直接进入 CLI 的 stdout
3. 删除临时文件
4. 若系统无 `pdftotext`：降级为打印预览 URL（`https://res.doclingo.ai/pdf/<task-id>.pdf`）
5. 若下载失败：返回 `not_found` 错误

## 输出示例

### 登录用户

```
正在下载到: translated_paper.pdf
OK: 下载完成: translated_paper.pdf
```

### 游客（pdftotext 可用）

```
游客模式 — 在控制台打印译文内容：
------------------------------------------------------------
<译文全文，包含布局保留的换行与空格>
...
------------------------------------------------------------
（如需保存为本地文件，请先执行 pdf-cli auth login --email you@example.com）
```

### 游客（pdftotext 不可用）

```
游客模式 — 在控制台打印译文内容：
------------------------------------------------------------
未检测到 pdftotext（poppler-utils），无法在控制台打印译文。
请在浏览器中打开预览 URL: https://res.doclingo.ai/pdf/<task-id>.pdf
------------------------------------------------------------
（如需保存为本地文件，请先执行 pdf-cli auth login --email you@example.com）
```

## API

- GET `user/operate/record/down/info?operateRecordId=<record-id>`（登录）
- HTTP GET 下载 PDF 本体：`<downloadBase>/pdf/<blobFileName>`

## 依赖

- **pdftotext**（来自 `poppler-utils`）：游客分支用于 PDF 文本提取
  - Linux: `apt install poppler-utils`
  - macOS: `brew install poppler`
  - 不安装时自动降级为打印预览 URL

## Notes

- `--task-id` 参数同时支持**数字 record-id** 和**字符串 task-id**：数字直接用作 record-id；否则走 `user/operate/record/list/page` 查表拿 record-id
- 多源下载机制避免单一 CDN 故障
- 游客分支不再返回 `AuthError`，而是把打印文本作为"结果"呈现，退出码 0
