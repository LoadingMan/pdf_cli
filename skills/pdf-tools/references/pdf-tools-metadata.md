# tools metadata

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

修改或移除 PDF 元数据。

## 命令

### 设置元数据

```bash
pdf-cli tools metadata set --file document.pdf --title "My Document" --author "Alice"
```

### 移除元数据

```bash
pdf-cli tools metadata remove --file document.pdf
```

## 参数

### metadata set

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--title <string>` | 否 | 文档标题 |
| `--author <string>` | 否 | 作者 |
| `--subject <string>` | 否 | 主题 |
| `--keywords <string>` | 否 | 关键词 |
| `--output <path>` | 否 | 输出文件路径 |

### metadata remove

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

设置元数据使用：

- `action: "EditPdfMetaData"`
- `pdfToolCode: 10`

移除元数据使用：

- `action: "RemovePdfMetaData"`
- `pdfToolCode: 11`

统一链路：

- 预上传：`POST core/tools/box/file/aws/pre/upload`
- 上传登记：`POST core/tools/box/file/new/upload`
- 提交任务：`POST core/tools/todo/operate`
- 查询状态：`GET core/tools/operate/status?queryKey=...`

## Notes

- 当前 metadata 参考仅覆盖已接入的子命令。
- 查询任务时使用 `queryKey`。
