# tools extract

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

从 PDF 中提取图片或文本。

## 命令

### 提取图片

```bash
pdf-cli tools extract image --file document.pdf

# 指定输出路径
pdf-cli tools extract image --file document.pdf --output images.zip
```

### 提取文本

```bash
pdf-cli tools extract text --file document.pdf

# 指定输出路径
pdf-cli tools extract text --file document.pdf --output content.txt
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

### extract image

- `action: "ExtractImage"`
- `pdfToolCode: 9`

### extract text

- `action: "CoverPDFTo"`
- `pdfToolCode: 13`
- `cover: "txt"`

统一链路：

- 预上传：`POST core/tools/box/file/aws/pre/upload`
- 上传登记：`POST core/tools/box/file/new/upload`
- 提交任务：`POST core/tools/todo/operate`
- 查询状态：`GET core/tools/operate/status?queryKey=...`

## Notes

- 提取文本虽然底层走 `CoverPDFTo`，但 CLI 命令归类在 `extract text`。
- 查询任务时使用 `queryKey`。
