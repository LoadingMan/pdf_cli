# tools split

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

按旧 `core/tools` 的拆分模式拆分 PDF。

## 命令

```bash
# 按固定页数拆分
pdf-cli tools split --file document.pdf --mode pages-per-pdf --pages-per-pdf 2

# 按奇偶页拆分
pdf-cli tools split --file document.pdf --mode even-odd

# 对半拆分
pdf-cli tools split --file document.pdf --mode cut-in-half

# 自定义拆分点
pdf-cli tools split --file document.pdf --mode custom --split-points 1,3,5

# 指定输出路径
pdf-cli tools split --file document.pdf --mode pages-per-pdf --pages-per-pdf 2 --output out.pdf
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--mode <type>` | 否 | 拆分模式：`pages-per-pdf`、`even-odd`、`cut-in-half`、`custom` |
| `--pages-per-pdf <int>` | 条件必填 | `pages-per-pdf` 模式下每个文件的页数 |
| `--split-points <string>` | 条件必填 | `custom` 模式下的拆分点，例如 `1,3,5` |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

- 预上传：`POST core/tools/box/file/aws/pre/upload`
- 上传登记：`POST core/tools/box/file/new/upload`
- 提交任务：`POST core/tools/todo/operate`
- 查询状态：`GET core/tools/operate/status?queryKey=...`

提交任务时使用：

- `action: "SplitFile"`
- `pdfToolCode: 5`
- `data.mode: pagesPerPdf | evenOdd | cutInHalf | custom`
- `data.pagesPerPdf` 或 `data.splitPoints`

## Notes

- 这条命令对应前端旧工具页的 `pagesPerPdf`、`evenOdd`、`cutInHalf`、`custom` 四种模式。
- `custom` 传的是拆分点，不是保留页码列表。
- 任务查询键是 `queryKey`。
