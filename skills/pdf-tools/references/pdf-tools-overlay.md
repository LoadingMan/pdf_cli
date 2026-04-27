# tools overlay

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

叠加两个 PDF。

## 命令

```bash
pdf-cli tools overlay --file base.pdf --overlay-file mark.pdf

# 以前景方式叠加
pdf-cli tools overlay --file base.pdf --overlay-file mark.pdf --position foreground

# 叠加页不足时重复最后一页
pdf-cli tools overlay --file base.pdf --overlay-file mark.pdf --repeat-last-overlay-page
```

## 参数

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | 底层 PDF 文件路径 |
| `--overlay-file <path>` | 是 | 叠加 PDF 文件路径 |
| `--position <string>` | 否 | 叠加位置，默认 `background`，可选 `background`、`foreground` |
| `--repeat-last-overlay-page` | 否 | 叠加页不足时重复最后一页 |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

- `action: "OverlayPDF"`
- `pdfToolCode: 15`
- 提交字段：`files`、`allPagesOverlay`、`repeatLastOverlayPage`、`overlayPosition`

统一链路：

- `POST core/tools/box/file/aws/pre/upload`
- `POST core/tools/box/file/new/upload`
- `POST core/tools/todo/operate`
- `GET core/tools/operate/status?queryKey=...`

## Notes

- 对应前端旧工具页的 PDF 叠加能力。
- 查询任务时使用 `queryKey`。
