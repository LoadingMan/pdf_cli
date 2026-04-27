---
name: pdf-tools
version: 1.0.0
description: "pdf-cli 工具模块：基于 core/tools 异步任务链路处理 PDF。支持合并、拆分、转换、旋转、压缩、水印、页面提取删除、页码、叠加、提取内容、元数据、安全设置与任务查询。处理 PDF 文件时使用。"
metadata:
  requires:
    bins: ["pdf-cli"]
  cliHelp: "pdf-cli tools --help"
---

# tools

**CRITICAL — 开始前 MUST 先用 Read 工具读取 [`../pdf-shared/SKILL.md`](../pdf-shared/SKILL.md)，其中包含认证、配置、输出与错误处理。**

## Core Concepts

- **baseURL**: 当前 tools 命令实际走主服务下的 `core/tools/*` 接口。
- **异步模型**: tools 命令提交任务后返回 `queryKey`，再查询状态或下载结果。
- **统一链路**: 预上传 -> 上传登记 -> 提交任务 -> 查询状态 -> 下载结果。
- **输出文件**: 大多数命令会在任务完成后自动下载结果，也可通过 `tools job download` 按 `queryKey` 单独下载。

## 异步任务流程

```text
1. POST core/tools/box/file/aws/pre/upload
2. POST core/tools/box/file/new/upload
3. POST core/tools/todo/operate
4. GET  core/tools/operate/status?queryKey=...
5. 下载结果
```

## 命令概览

### 基础操作

| 命令 | 说明 |
| ------ | ------ |
| [`merge`](references/pdf-tools-merge.md) | 合并多个 PDF 文件 |
| [`split`](references/pdf-tools-split.md) | 按指定模式拆分 PDF |
| [`reorder`](references/pdf-tools-reorder.md) | 重排 PDF 页面顺序 |
| [`rotate`](references/pdf-tools-rotate.md) | 旋转指定页面 |
| [`compress`](references/pdf-tools-compress.md) | 压缩 PDF 文件 |
| [`watermark`](references/pdf-tools-watermark.md) | 添加文字水印 |
| [`overlay`](references/pdf-tools-overlay.md) | 叠加两个 PDF |

### 转换与提取

| 命令 | 说明 |
| ------ | ------ |
| [`convert pdf-to-word`](references/pdf-tools-convert.md) | PDF 转 Word |
| [`extract image`](references/pdf-tools-extract.md) | 提取 PDF 中的图片 |
| [`extract text`](references/pdf-tools-extract.md) | 提取 PDF 中的文本 |

### 页面操作

| 命令 | 说明 |
| ------ | ------ |
| [`page extract`](references/pdf-tools-page.md) | 提取指定页面为新 PDF |
| [`page delete`](references/pdf-tools-page.md) | 删除指定页面 |
| [`page-number add`](references/pdf-tools-page-number.md) | 添加页码 |

### 安全

| 命令 | 说明 |
| ------ | ------ |
| [`security encrypt`](references/pdf-tools-security.md) | 加密 PDF |
| [`security decrypt`](references/pdf-tools-security.md) | 解密 PDF |

### 元数据

| 命令 | 说明 |
| ------ | ------ |
| [`metadata set`](references/pdf-tools-metadata.md) | 修改 PDF 元数据 |
| [`metadata remove`](references/pdf-tools-metadata.md) | 移除 PDF 元数据 |

### 任务管理

| 命令 | 说明 |
| ------ | ------ |
| [`job status`](references/pdf-tools-job.md) | 查询任务状态 |
| [`job download`](references/pdf-tools-job.md) | 下载任务结果 |

## Important Notes

- 当前 CLI 仅暴露文档中列出的 tools 子命令。
- 当前 CLI 的转换命令只暴露 `convert pdf-to-word`，提取文本使用 `extract text`。
- 任务查询主参数是 `--query-key`，`--job-id` 仅保留兼容。
- skills 文档以当前 CLI 命令树和 `core/tools/*` 为准。
