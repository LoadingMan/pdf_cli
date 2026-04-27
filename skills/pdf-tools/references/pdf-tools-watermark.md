# tools watermark

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

为 PDF 添加文字水印。

## 命令

```bash
pdf-cli tools watermark --file document.pdf --text "CONFIDENTIAL"

# 指定颜色、透明度和位置
pdf-cli tools watermark --file document.pdf --text "DRAFT" --color "#999999" --alpha 0.25 --position center-center --angle -45

# 指定输出路径
pdf-cli tools watermark --file document.pdf --text "DRAFT" --output watermarked.pdf
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--text <string>` | 是 | 水印文字 |
| `--font-size <int>` | 否 | 字体大小，默认 `40` |
| `--color <string>` | 否 | 颜色，默认 `#000000` |
| `--alpha <string>` | 否 | 透明度，默认 `0.4` |
| `--position <string>` | 否 | 位置，默认 `center-center` |
| `--angle <string>` | 否 | 旋转角度，默认 `-45` |
| `--space-x <string>` | 否 | 水平间距，默认 `5` |
| `--space-y <string>` | 否 | 垂直间距，默认 `5` |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

- `action: "AddWatermark"`
- `pdfToolCode: 17`
- 提交字段：`pattern`、`position`、`fontFamily`、`fontWeight`、`fontStyle`、`fontSize`、`color`、`alpha`、`angle`、`spaceX`、`spaceY`

统一链路：

- 预上传：`POST core/tools/box/file/aws/pre/upload`
- 上传登记：`POST core/tools/box/file/new/upload`
- 提交任务：`POST core/tools/todo/operate`
- 查询状态：`GET core/tools/operate/status?queryKey=...`

## Notes

- 对应前端旧工具页的文字水印能力。
- 查询任务时使用 `queryKey`。
