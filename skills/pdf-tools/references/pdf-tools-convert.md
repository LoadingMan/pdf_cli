# tools convert

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

PDF 格式转换。当前 CLI 只暴露 `pdf-to-word`。

## 命令

```bash
pdf-cli tools convert pdf-to-word --file document.pdf

# 指定输出路径
pdf-cli tools convert pdf-to-word --file document.pdf --output result.docx
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

- `action: "CoverPDFTo"`
- `pdfToolCode: 2`
- `data.cover: "docx"`

统一链路：

- `POST core/tools/box/file/aws/pre/upload`
- `POST core/tools/box/file/new/upload`
- `POST core/tools/todo/operate`
- `GET core/tools/operate/status?queryKey=...`

## Notes

- 提取文本不走 `convert`，使用 `pdf-cli tools extract text`。
- 当前 `convert` 参考仅覆盖已接入的子命令。
- 查询任务时使用 `queryKey`。
