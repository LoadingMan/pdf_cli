# tools page-number

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

为 PDF 添加页码。

## 命令

```bash
pdf-cli tools page-number add --file document.pdf

# 自定义格式和位置
pdf-cli tools page-number add --file document.pdf --pattern "{NUM}" --position bottom-center

# 自定义样式
pdf-cli tools page-number add --file document.pdf --font-family serif --font-size 10 --font-weight bold --font-style normal --color "#666666"
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--pattern <string>` | 否 | 页码格式，默认 `"{NUM}/{CNT}"` |
| `--position <string>` | 否 | 位置，默认 `bottom-right` |
| `--font-family <string>` | 否 | 字体族，默认 `sans` |
| `--font-size <string>` | 否 | 字体大小，默认 `8` |
| `--font-weight <string>` | 否 | 字重，默认 `normal` |
| `--font-style <string>` | 否 | 字形，默认 `italic` |
| `--color <string>` | 否 | 颜色，默认 `#000000` |
| `--alpha <string>` | 否 | 透明度，默认 `0.8` |
| `--angle <string>` | 否 | 旋转角度，默认 `0` |
| `--space-x <string>` | 否 | 水平间距，默认 `5` |
| `--space-y <string>` | 否 | 垂直间距，默认 `5` |
| `--page-num-offset <string>` | 否 | 起始页码偏移，默认 `0` |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

- `action: "AddPageNumbers"`
- `pdfToolCode: 7`
- 提交字段：`pattern`、`position`、`fontFamily`、`fontSize`、`fontWeight`、`fontStyle`、`color`、`alpha`、`angle`、`spaceX`、`spaceY`、`pageNumOffset`

统一链路：

- `POST core/tools/box/file/aws/pre/upload`
- `POST core/tools/box/file/new/upload`
- `POST core/tools/todo/operate`
- `GET core/tools/operate/status?queryKey=...`

## Notes

- 对应前端旧工具页的页码功能。
- 查询任务时使用 `queryKey`。
