# tools rotate

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

旋转指定 PDF 页面。

## 命令

```bash
# 旋转指定页 90 度
pdf-cli tools rotate --file document.pdf --pages 1,3 --angle 90

# 旋转指定页 180 度
pdf-cli tools rotate --file document.pdf --pages 2,4 --angle 180

# 指定输出路径
pdf-cli tools rotate --file document.pdf --pages 1,2 --angle 270 --output rotated.pdf
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--pages <string>` | 是 | 需要旋转的页码列表，例如 `1,3,5` |
| `--angle <int>` | 否 | 旋转角度：`90`、`180`、`270` |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

- 预上传：`POST core/tools/box/file/aws/pre/upload`
- 上传登记：`POST core/tools/box/file/new/upload`
- 提交任务：`POST core/tools/todo/operate`
- 查询状态：`GET core/tools/operate/status?queryKey=...`

提交任务时使用：

- `action: "RotatePDFPages"`
- `pdfToolCode: 16`
- `data.rotateInfo: [{ file, rotate }]`

## Notes

- 当前 CLI 仅支持显式页码列表，不支持 `all`。
- `-90`、`-180`、`-270` 会在内部归一化为 `270`、`180`、`90`。
- 任务查询键是 `queryKey`。
