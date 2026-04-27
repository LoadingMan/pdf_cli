# tools security

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

PDF 加密与解密操作。

## 命令

### 加密

```bash
pdf-cli tools security encrypt --file document.pdf --password mypassword

# 指定权限
pdf-cli tools security encrypt --file document.pdf --password mypassword --allow-print=false --allow-extract=false

# 指定输出路径
pdf-cli tools security encrypt --file document.pdf --password mypassword --output encrypted.pdf
```

### 解密

```bash
pdf-cli tools security decrypt --file encrypted.pdf --password mypassword

# 指定输出路径
pdf-cli tools security decrypt --file encrypted.pdf --password mypassword --output decrypted.pdf
```

## 参数

### encrypt

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--password <string>` | 是 | 用户密码 |
| `--allow-assemble` | 否 | 允许组装文档 |
| `--allow-extract` | 否 | 允许提取内容 |
| `--allow-accessibility` | 否 | 允许辅助功能提取 |
| `--allow-fill-form` | 否 | 允许填写表单 |
| `--allow-modify` | 否 | 允许修改内容 |
| `--allow-annotate` | 否 | 允许修改注释 |
| `--allow-print` | 否 | 允许打印 |
| `--allow-print-hq` | 否 | 允许高质量打印 |
| `--output <path>` | 否 | 输出文件路径 |

### decrypt

| 参数 | 必填 | 说明 |
| ------ | ------ | ------ |
| `--file <path>` | 是 | PDF 文件路径 |
| `--password <string>` | 是 | 当前密码 |
| `--output <path>` | 否 | 输出文件路径 |

## 接口映射

加密使用：

- `action: "LockPDF"`
- `pdfToolCode: 3`
- 使用 `userPass` 和多个权限字段提交

解密使用：

- `action: "UnlockPDF"`
- `pdfToolCode: 4`
- 使用 `userPass` 提交

统一链路：

- 预上传：`POST core/tools/box/file/aws/pre/upload`
- 上传登记：`POST core/tools/box/file/new/upload`
- 提交任务：`POST core/tools/todo/operate`
- 查询状态：`GET core/tools/operate/status?queryKey=...`

## Notes

- 对应前端旧工具页的加密与解密能力。
- 查询任务时使用 `queryKey`。
