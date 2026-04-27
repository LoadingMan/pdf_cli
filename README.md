# pdf-cli

[![License](https://img.shields.io/badge/License-UNLICENSED-lightgrey.svg)](#许可证)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.17-blue.svg)](https://go.dev/)
[![npm version](https://img.shields.io/npm/v/@loadingman/pdf_cli.svg)](https://www.npmjs.com/package/@loadingman/pdf_cli)

PDF 翻译与 PDF 工具命令行客户端。覆盖 PDF 翻译、合并、转换、拆分、压缩、加密、页面操作、内容提取等核心场景。

[安装](#安装与快速开始) · [认证](#认证) · [命令总览](#命令总览) · [高级用法](#高级用法) · [配置](#配置) · [详细文档](#详细文档)

## 为什么用 pdf-cli？

- **覆盖广** — 翻译、工具、用户、会员四大模块，60+ 子命令，对应官方接口
- **异步友好** — tools 全链路自动轮询任务状态，超时仍返回 `queryKey` 便于后续手动恢复
- **多种输出** — `--format json/pretty/table` 三种格式，错误统一为 JSON envelope，方便脚本接管
- **零依赖二进制** — Go 静态编译，无运行时依赖，单文件即可使用
- **两种安装方式** — npm 一键安装预编译二进制，或从源码 `make install`

## 功能模块

| 模块            | 能力                                                                                                       |
| --------------- | ---------------------------------------------------------------------------------------------------------- |
| 🌐 translate    | PDF 文档翻译、AI 文本翻译、arXiv 取译、引擎/语种枚举、任务状态轮询、历史记录                                |
| 🛠 tools        | 合并、PDF 转 Word、拆分、重排、旋转、页面提取/删除、页码、压缩、水印、叠加、提取图片/文本、元数据、加解密 |
| 🔐 auth         | 邮箱密码登录、登录状态、登出                                                                              |
| 👤 user         | 个人资料、上传文件、使用记录、API Key 管理、反馈提交                                                       |
| 💎 member       | 会员信息、权益、定价、订单、兑换码                                                                         |
| 📢 other        | 公告、版本历史、使用指南                                                                                  |

## 安装与快速开始

### 环境要求

开始之前请确认：

- Node.js（`npm` / `npx`）—— 仅 npm 安装方式需要
- Go `v1.17`+ —— 仅从源码安装时需要

### 安装

从以下两种方式中任选其一：

**方式一 — 从 npm 安装（推荐）：**

```bash
npm install -g @loadingman/pdf_cli

# 安装 CLI SKILL（必需）
npx skills add @loadingman/pdf_cli -y -g
```

`postinstall` 会根据当前平台（darwin / linux / windows × x64 / arm64）自动从 GitHub Release 下载对应的预编译二进制，并把 `pdf-cli` 命令注册到 PATH。

**方式二 — 从源码安装：**

需要 Go `v1.17`+。

```bash
git clone https://gitee.com/loadingmans/pdf_cli.git
cd pdf_cli
make install

# 安装 CLI SKILL（必需）
npx skills add @loadingman/pdf_cli -y -g
```

`make install` 会编译并安装到 `/usr/local/bin/pdf-cli`。如需自定义路径：

```bash
make install PREFIX=$HOME/.local
```

### 配置与使用

```bash
# 1. 登录（密码以隐藏方式交互输入）
pdf-cli auth login --email you@example.com

# 2. 翻译 PDF（异步：上传 → 启动 → 轮询 → 下载）
pdf-cli translate upload --file ./paper.pdf
pdf-cli translate start --file-key <key> --to zh
pdf-cli translate status --task-id <id> --wait
pdf-cli translate download --task-id <id> -o paper.zh.pdf

# 3. PDF 工具
pdf-cli tools merge --files a.pdf,b.pdf,c.pdf
pdf-cli tools convert pdf-to-word --file document.pdf
pdf-cli tools page extract --file document.pdf --pages 1-3
pdf-cli tools job status --query-key <key>
```

执行 `pdf-cli --help` 查看所有命令。

## 认证

| 命令          | 说明                       |
| ------------- | -------------------------- |
| `auth login`  | 邮箱密码登录，密码隐藏输入 |
| `auth status` | 查看当前登录身份与 token   |
| `auth logout` | 注销并清除本地凭证         |

```bash
pdf-cli auth login --email you@example.com
pdf-cli auth status
pdf-cli auth logout
```

凭证保存在 `~/.config/pdf-cli/config.json`（权限 `0600`），包含 `token` 和稳定的 `device_id`。

## 命令总览

### translate — 翻译

```bash
pdf-cli translate languages
pdf-cli translate engines
pdf-cli translate upload --file ./paper.pdf
pdf-cli translate start --file-key <key> --to zh
pdf-cli translate start --file-key <key> --to zh --from en
pdf-cli translate free --file ./paper.pdf --to zh
pdf-cli translate arxiv --arxiv-id 2401.00001 --to zh
pdf-cli translate text --text "hello world" --to zh
pdf-cli translate status --task-id <id>
pdf-cli translate status --task-id <id> --wait
pdf-cli translate download --task-id <id> -o result.pdf
pdf-cli translate history
pdf-cli translate history --page 2 --page-size 10
```

### tools — PDF 工具

```bash
# 合并
pdf-cli tools merge --files a.pdf,b.pdf,c.pdf
pdf-cli tools merge --files a.pdf,b.pdf --create-bookmarks

# 转换
pdf-cli tools convert pdf-to-word --file document.pdf

# 拆分
pdf-cli tools split --file document.pdf --mode pages-per-pdf --pages-per-pdf 2
pdf-cli tools split --file document.pdf --mode even-odd
pdf-cli tools split --file document.pdf --mode cut-in-half
pdf-cli tools split --file document.pdf --mode custom --split-points 2,5

# 页面操作
pdf-cli tools reorder --file document.pdf --order 3,1,2
pdf-cli tools rotate --file document.pdf --pages 1,2 --angle 90
pdf-cli tools page extract --file document.pdf --pages 1-3,5
pdf-cli tools page delete --file document.pdf --pages 2,4
pdf-cli tools page-number add --file document.pdf --pattern "{NUM}/{CNT}"

# 压缩与水印
pdf-cli tools compress --file large.pdf --dpi 144 --image-quality 75
pdf-cli tools compress --file large.pdf --color-mode gray
pdf-cli tools watermark --file document.pdf --text "CONFIDENTIAL"
pdf-cli tools overlay --file base.pdf --overlay-file overlay.pdf --position foreground

# 提取内容
pdf-cli tools extract image --file document.pdf
pdf-cli tools extract text --file document.pdf

# 元数据
pdf-cli tools metadata set --file document.pdf --title "My Doc" --author "Alice"
pdf-cli tools metadata remove --file document.pdf

# 加密 / 解密
pdf-cli tools security encrypt --file document.pdf --password 123456
pdf-cli tools security decrypt --file encrypted.pdf --password 123456

# 异步任务管理
pdf-cli tools job status --query-key <key>
pdf-cli tools job download --query-key <key> -o result.pdf
```

### user — 用户

```bash
pdf-cli user profile
pdf-cli user update --name Alice
pdf-cli user files list
pdf-cli user records list
pdf-cli user records get --id 101
pdf-cli user api-key list
pdf-cli user api-key create --name "my-key"
pdf-cli user api-key delete --id 1
pdf-cli user feedback submit --title "标题" --content "内容"
```

### member — 会员

```bash
pdf-cli member info
pdf-cli member rights
pdf-cli member pricing
pdf-cli member order list
pdf-cli member order get --order-no ORD123
pdf-cli member redeem --code XXXX-XXXX-XXXX
```

### other — 其他

```bash
pdf-cli other version
pdf-cli other notice
pdf-cli other help-guide
```

## 高级用法

### 输出格式

全局 `--format` 标志影响所有命令：

```bash
--format pretty    # 人类友好对齐输出（默认）
--format json      # 完整 JSON 响应，错误也包成 JSON envelope
--format table     # ASCII 表格
```

```bash
pdf-cli user profile --format json
pdf-cli translate history --format table
```

### 输出到文件

```bash
pdf-cli translate download --task-id <id> -o ./out/result.pdf
pdf-cli user profile --format json --output profile.json
```

`--output` 既可指向文件，也可指向目录（自动追加默认文件名）。

### 异步任务链路

tools 子命令底层都是异步任务，CLI 会自动轮询直到完成或超时（默认每 2 秒一次，最多 120 次 ≈ 4 分钟）。超时不会丢失任务，会打印 `queryKey` 让你之后用 `tools job status / download` 继续：

```text
1. POST core/tools/box/file/aws/pre/upload      # 申请预签名上传
2. POST core/tools/box/file/new/upload          # 注册上传记录
3. POST core/tools/todo/operate                 # 启动操作，得到 queryKey
4. GET  core/tools/operate/status?queryKey=...  # 轮询状态
5. 下载结果文件
```

### 退出码

| Code | 含义                       |
| ---- | -------------------------- |
| 0    | 成功                       |
| 2    | 用法错误（参数解析失败）   |
| 3    | 参数校验失败               |
| 11   | 未登录或登录态失效         |
| 14   | 任务执行失败               |

便于 shell 脚本根据退出码做不同分支。

## 配置

配置文件位于 `~/.config/pdf-cli/config.json`，目录权限 `0700`，文件权限 `0600`：

```json
{
  "token": "登录后自动保存",
  "device_id": "首次登录自动生成的 UUID",
  "base_url": "主服务地址",
  "tool_url": "工具服务地址",
  "download_url": "文件下载基地址",
  "format": "默认输出格式（pretty/json/table）"
}
```

`device_id` 在 `auth logout` 后仍会保留，避免设备指纹漂移。

## 详细文档

各模块完整命令、参数说明、接口映射见 [document/](./document/) 目录：

| 文档                                          | 说明                          |
| --------------------------------------------- | ----------------------------- |
| [index.md](./document/index.md)               | 文档总入口                    |
| [cli-report.md](./document/cli-report.md)     | 完整 CLI 行为参考             |
| [api-mapping.md](./document/api-mapping.md)   | CLI 命令到后端接口的映射      |
| [async-flows.md](./document/async-flows.md)   | 异步任务与 queryKey 流程      |
| [end-to-end.md](./document/end-to-end.md)     | 端到端示例工作流              |
| [errors.md](./document/errors.md)             | 错误码与异常处理              |
| [conventions.md](./document/conventions.md)   | 代码与命令命名约定            |
| [proposal.md](./document/proposal.md)         | 设计提案与决策记录            |

## 维护者发布流程

仅项目维护者需要关注。完整链路在 [scripts/](./scripts/)：

```bash
# 1. bump package.json 版本，提交并推送
# 2. 打 tag（脚本会做前置检查）
bash scripts/tag-release.sh

# 3. 交叉编译 6 个平台二进制到 dist/
bash scripts/release.sh

# 4. 把 dist/ 下产物上传到对应 GitHub Release
gh release upload "$(node -p "require('./package.json').version")" dist/pdf-cli-*.{tar.gz,zip}

# 5. 发布 npm 包
npm publish --access public
```

注意：当前 tag 命名约定是裸版本号（`1.0.0`），不带 `v` 前缀，与 [scripts/install.js](./scripts/install.js) 拼接的 Release URL 保持一致。

## 说明

- 当前 CLI 仅暴露 README 中列出的 tools 子命令。
- `convert` 当前只暴露 `pdf-to-word`；提取文本请使用 `extract text`。
- `tools job status` 与 `tools job download` 主参数为 `--query-key`，`--job-id` 仅保留兼容。

## 许可证

UNLICENSED — 内部使用。详见 [package.json](./package.json)。
