# translate text

翻译指定文本内容到目标语言。

## 用法

```bash
pdf-cli translate text --record-id <id> --text "待翻译文本" --engine <引擎ID>
pdf-cli translate text --record-id 123 --text "Hello World" --engine 1
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--record-id` | 是 | 操作记录 ID |
| `--text` | 是 | 待翻译的文本内容 |
| `--engine` | 是 | 翻译引擎 ID（通过 `pdf-cli translate engines` 查看） |

## API

- **端点**: `POST core/pdf/trans/text/area`
- **认证**: 需要登录

## 响应字段

| 字段 | 说明 |
|------|------|
| `translatedText` | 翻译后的文本 |

## 注意

- 需要先有一个翻译任务的 record-id
- 会消耗用户字符配额
