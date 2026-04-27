# api-mapping

CLI 命令到后端 HTTP 接口的映射。本文件给需要绕过 CLI 直接打后端的 agent 使用，或者用于排错时定位是哪个接口出了问题。

> **服务端基址**：默认 `https://api.gdpdf.com/api`，可在 `~/.config/pdf-cli/config.json` 的 `base_url` 字段覆盖。所有相对路径都拼在该基址之后。
>
> **请求头**：base API 使用 `token: <token>`、`deviceId: <id>`、`clientType: cli`、`internationalCode: zh-CN`；tool API 使用 `X-API-KEY: <token>`。
>
> **响应包络**：base API 使用 `{code, data, message, list, dataList}`，`code == 1` 或 `code == 200` 表示成功；tool API 使用 `{code, data, state, message}`，state 取大写值（SUCCESS/FAILURE/PROCESSING）。

## auth

| CLI 命令 | 方法 | 路径 | 鉴权 |
|---|---|---|---|
| `auth login` | POST | `user/basic/base/login` | 否（产生 token） |
| `auth status` | GET | `user/basic/token/userinfo` | token |
| `auth logout` | GET | `user/basic/login/out` | token |

请求体（login）：`{"userEmail": "...", "password": "..."}`，响应 `data` 含 token 字符串或对象 `{token: "..."}`。

## translate

### 元信息查询

| CLI 命令 | 方法 | 路径 | 鉴权 |
|---|---|---|---|
| `translate languages` | GET | `core/pdf/lang/list` | 否 |
| `translate engines` | GET | `core/pdf/engines` | 否 |
| `translate arxiv-info` | GET | `core/pdf/query/arxiv/summary` | 否 |

### 上传

`translate upload` 是三步链路：

| 步骤 | 方法 | 路径 | 说明 |
|---|---|---|---|
| 1 | POST | `core/pdf/trans/aws/pre/upload` | 取 AWS 预签名 URL，参数 `{fileRealName, freeTag}` |
| 2 | PUT | `<awsUploadUrl>` | 直接上传到 S3，Content-Type: application/pdf，独立 10 分钟超时 |
| 3 | POST | `core/pdf/file/new/upload` | 注册上传，参数 `{blobFileName, fileMd5String, fileRealName, fileSize, freeTag}` |

输出：`tmpFileName`（即 `file-key`）、`sourceFileId`。

### 发起翻译

| CLI 命令 | 方法 | 路径 | 鉴权 |
|---|---|---|---|
| `translate start` | POST | `core/pdf/translate` | token |
| `translate free` | POST | `core/pdf/free/translate` | 否 |
| `translate arxiv` | POST | `core/pdf/arxiv/translate` | 否 |

请求体字段（start，其他类似）：

```
{
  "fileKey":         "<file-key>",
  "targetLang":      "zh",
  "sourceLang":      "en",            // 可选
  "transEngineType": "google",        // 可选
  "ocrFlag":         0,               // 0 或 1
  "termIds":         "",              // 可选，术语库 ID 列表
  "fileFmtType":     "",              // 可选
  "promptType":      0                // 可选
}
```

输出：`blobFileName`（去扩展名 = `task-id`）、`id`（= `record-id`）。

### 任务管理

| CLI 命令 | 方法 | 路径 | 鉴权 | 关键参数 |
|---|---|---|---|---|
| `translate status` | GET | `core/pdf/query/status` | 否 | `queryFileKey=<task-id>` |
| `translate continue` | POST | `core/pdf/trans/continue` | token | `{operateRecordId}` |
| `translate cancel` | POST | `core/pdf/cancel/trans` | token | `{operateRecordId}` |
| `translate history` | POST | `user/operate/record/list/page` | token | `{pageNo, pageSize}` |

`continue` / `cancel` 接受的是 record-id（数字）。CLI 支持传 task-id，会先用 history 接口反查 record-id（见 `translateResolveRecordID`）。

### 下载

| CLI 命令 | 方法 | 路径 | 鉴权 |
|---|---|---|---|
| `translate download` | GET | `user/operate/record/down/info` | 可选 |
| `translate download` (回退) | GET | `<base>/pdf/<blobFileName>` | 否 |

下载源回退顺序（CLI 内部已实现）：
1. 已登录：先调 `user/operate/record/down/info` 取下载 URL
2. 静态回退 1：`https://api.gdpdf.com/pdf/<blobFileName>`
3. 静态回退 2：`https://res.gdpdf.com/pdf/<blobFileName>`

所有源失败 → exit 8 (not_found)。

### 文本流式翻译

| CLI 命令 | 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|---|
| `translate text` | POST | `core/ai/translate/askstream` | 否（可选 token） | SSE 流响应 |

请求头：`Accept: text/event-stream`。响应是 SSE，事件 `[DATA]` 包含 `{text: "..."}` 增量；`[ERROR]` 包含错误消息；`[FINISH]` 表示结束。CLI 内部把 `[ERROR]` 映射为 exit 14 (task_failed)。

## tools

所有 tools 命令共享同一条异步链路（详见 [async-flows.md](async-flows.md) §2）。

### 链路接口

| 步骤 | 方法 | 路径 | 鉴权 |
|---|---|---|---|
| 上传准备 | POST | `core/tools/box/file/aws/pre/upload` | token |
| AWS 上传 | PUT | `<awsUploadUrl>` | 否（预签名 URL） |
| 上传登记 | POST | `core/tools/box/file/new/upload` | token |
| 提交任务 | POST | `core/tools/todo/operate` | token |
| 状态轮询 | GET | `core/tools/operate/status` | token |
| 结果下载 | GET | `<base>/pdf/box/<filename>` | 否 |

### `core/tools/todo/operate` 请求体

```
{
  "action":      "<action-name>",   // merge / split / convert / ...
  "data":        { ... },            // 工具特定的参数
  "pdfToolCode": <int>               // 工具数字编码
}
```

`pdfToolCode` 对照表（不完整，由源码硬编码，详见 [cmd/tools_cmds.go](../cmd/tools_cmds.go)）：

| 工具 | pdfToolCode |
|---|---|
| merge | 1 |
| split | 2 |
| pdf-to-word | 3 |
| extract image | 4 |
| extract text | 5 |
| compress | 6 |
| watermark | 7 |
| rotate | 8 |
| reorder | 9 |
| page extract | 10 |
| page delete | 11 |
| page-number add | 12 |
| metadata set | 13 |
| metadata remove | 14 |
| security encrypt | 15 |
| security decrypt | 16 |
| overlay | 17 |

> 上表是大致顺序，具体值以 [cmd/tools_cmds.go](../cmd/tools_cmds.go) / [cmd/tools_extra.go](../cmd/tools_extra.go) 中的 `pdfToolCode` 常量为准。

### `tools job *`

| CLI 命令 | 方法 | 路径 | 鉴权 |
|---|---|---|---|
| `tools job status` | GET | `core/tools/operate/status?queryKey=<key>` | token |
| `tools job download` | GET (轮询) + 静态下载 | 同上 + `<base>/pdf/box/<filename>` | token |

下载源回退顺序：
1. 默认：`<base>/pdf/box/<filename>`，base 取 config 中的 `tool_url` 或 `base_url`
2. 回退 1（自动主机替换）：将 `gdpdf.com` 替换为 `res.gdpdf.com`，`pre.gdpdf.com` 替换为 `res.pre.gdpdf.com`

## user

| CLI 命令 | 方法 | 路径 | 鉴权 |
|---|---|---|---|
| `user profile` | GET | `user/basic/token/userinfo` | token |
| `user update` | POST | `user/basic/modify/userinfo` | token |
| `user files list` | POST | `user/source/file/list` | token |
| `user records list` | POST | `user/operate/record/list/page` | token |
| `user records get` | GET | `user/operate/record/get` | token |
| `user api-key list` | GET | `user/apicommon/sk/list` | token |
| `user api-key create` | POST | `user/apicommon/sk/add` | token |
| `user api-key delete` | POST | `user/apicommon/sk/del` | token |
| `user feedback submit` | POST | `user/feedback/save` | token |

## member

| CLI 命令 | 方法 | 路径 | 鉴权 |
|---|---|---|---|
| `member info` | GET | `user/config/vip/cfg` | token |
| `member rights` | GET | `user/basic/get/vip/functions` | token |
| `member pricing` | GET | `user/config/all/price/cfg` | token |
| `member order list` | POST | `user/trade/list` | token |
| `member order get` | GET | `user/trade/get` | token |
| `member redeem` | POST | `user/vipcode/bind` | token |

## other

| CLI 命令 | 方法 | 路径 | 鉴权 |
|---|---|---|---|
| `other version` | GET | `user/history/version/list?clientType=1` | 否 |
| `other notice` | – | – | – |
| `other help-guide` | GET | `user/config/homepage` | 否 |

## 直接调后端的注意事项

如果 agent 选择绕过 CLI 直接打后端：

1. **必须带请求头** `clientType` 和 `internationalCode: zh-CN`。后端会因为缺失这些头返回 "Parameter error"。
2. **deviceId 必须是合法 UUID**，建议从 CLI 配置 `~/.config/pdf-cli/config.json` 读，或自己生成一个并保持稳定。
3. **token 是 base API 用 `token` header，tool API 用 `X-API-KEY` header**。混用会得到 401。
4. **所有 base API 的成功判定**：`code == 1` 或 `code == 200`，类型可能是字符串或数字。
5. **后端业务码不等于 HTTP 状态码**：HTTP 200 + `code == 400` 是合法的"业务参数错误"响应，CLI 把它映射成 exit 3 (invalid_argument)。直接调后端的 agent 需要自己实现这层映射，可参考 [internal/client/client.go](../internal/client/client.go) 的 `classifyBusinessError`。
