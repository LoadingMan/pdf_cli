# tools page

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

页面提取与删除。

## 命令

### 提取指定页面

```bash
pdf-cli tools page extract --file document.pdf --pages 1-3,5
```

### 删除指定页面

```bash
pdf-cli tools page delete --file document.pdf --pages 2,4
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--pages <string>` | 是 | 页码范围，例如 `1-3,5` |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

提取页面使用：

- `action: "ExtractPdfPages"`
- `pdfToolCode: 12`
- `data.extractInfo: [{ filename, name, fileIndex, pageIndex }]`

删除页面使用：

- `action: "RemovePDFPages"`
- `pdfToolCode: 8`
- `data.removeInfo: [{ filename, name, fileIndex, pageIndex }]`

统一链路：

- `POST core/tools/box/file/aws/pre/upload`
- `POST core/tools/box/file/new/upload`
- `POST core/tools/todo/operate`
- `GET core/tools/operate/status?queryKey=...`

## Notes

- `--pages` 支持单页和范围混写。
- CLI 会把页码转成后端需要的页面选择结构。
- 查询任务时使用 `queryKey`。
