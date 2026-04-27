# tools compress

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

按旧工具链参数压缩 PDF。

## 命令

```bash
# 使用默认参数压缩
pdf-cli tools compress --file document.pdf

# 指定 DPI 和图片质量
pdf-cli tools compress --file document.pdf --dpi 144 --image-quality 75

# 转灰度压缩
pdf-cli tools compress --file document.pdf --grayscale
pdf-cli tools compress --file document.pdf --color-mode gray

# 指定输出路径
pdf-cli tools compress --file document.pdf --output compressed.pdf
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--dpi <int>` | 否 | DPI，默认 `144` |
| `--image-quality <int>` | 否 | 图片质量，默认 `75` |
| `--grayscale` | 否 | 兼容开关，压缩时转灰度 |
| `--color-mode <type>` | 否 | 颜色模式：`color`、`gray` |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

- 预上传：`POST core/tools/box/file/aws/pre/upload`
- 上传登记：`POST core/tools/box/file/new/upload`
- 提交任务：`POST core/tools/todo/operate`
- 查询状态：`GET core/tools/operate/status?queryKey=...`

提交任务时使用：

- `action: "CompressPDF"`
- `pdfToolCode: 14`
- `data.dpi`
- `data.imageQuality`
- `data.colorMode: "Gray"`（当使用灰度模式时）

## Notes

- 这条命令对应前端旧工具页的压缩参数：`dpi`、`imageQuality`、`colorMode`。
- `--color-mode` 优先于 `--grayscale`。
- 任务查询键是 `queryKey`。
