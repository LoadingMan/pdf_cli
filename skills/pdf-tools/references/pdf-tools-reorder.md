# tools reorder

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

重排 PDF 页面顺序。

## 命令

```bash
pdf-cli tools reorder --file document.pdf --order 3,1,2

# 指定输出路径
pdf-cli tools reorder --file document.pdf --order 3,1,2 --output reordered.pdf
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--order <string>` | 是 | 页面顺序，例如 `3,1,2` |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

- `action: "RearrangePDFPages"`
- `pdfToolCode: 6`
- `data.sortInfo: [{ file, pages }]`

统一链路：

- 预上传：`POST core/tools/box/file/aws/pre/upload`
- 上传登记：`POST core/tools/box/file/new/upload`
- 提交任务：`POST core/tools/todo/operate`
- 查询状态：`GET core/tools/operate/status?queryKey=...`

## Notes

- 对应前端旧工具页的页面重排能力。
- 查询任务时使用 `queryKey`。
