# user feedback

> **前置条件：** 先阅读 [`../pdf-shared/SKILL.md`](../../pdf-shared/SKILL.md)。

提交意见反馈。

## 命令

```bash
pdf-cli user feedback submit --title "功能建议" --content "希望支持批量翻译"
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--title <string>` | 是 | 反馈标题 |
| `--content <string>` | 是 | 反馈内容 |

## API

- POST JSON `user/feedback/save`
- 请求体：`{title, content}`
- 需要登录
