# tools merge

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

合并多个 PDF 文件为一个。

## 命令

```bash
# 合并两个文件
pdf-cli tools merge --files a.pdf,b.pdf

# 多次指定 --files
pdf-cli tools merge --files a.pdf --files b.pdf --files c.pdf

# 合并时创建书签
pdf-cli tools merge --files a.pdf,b.pdf --create-bookmarks

# 指定输出路径
pdf-cli tools merge --files a.pdf,b.pdf --output merged.pdf
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--files <paths>` | 是 | PDF 文件路径列表，可用逗号分隔或多次指定 |
| `--create-bookmarks` | 否 | 合并时创建书签 |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

- 预上传：`POST core/tools/box/file/aws/pre/upload`
- 上传登记：`POST core/tools/box/file/new/upload`
- 提交任务：`POST core/tools/todo/operate`
- 查询状态：`GET core/tools/operate/status?queryKey=...`
- 下载结果：`GET {download_url}/pdf/box/{filename}`

提交任务时使用：

- `action: "CombineFile"`
- `pdfToolCode: 1`
- `data.files: [{ filename, name }]`
- `data.createBookmarks: true|false`

## Notes

- 这条命令对应前端旧工具页的 PDF Merge。
- 命令提交后会自动轮询，成功后直接下载结果文件。
- 任务查询键是 `queryKey`，不是 `jobId`。
