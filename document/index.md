# pdf-cli document

pdf-cli 的 **agent 集成参考文档**。本目录的内容是给"调用 pdf-cli 的 agent 实现者"用的，不是给最终用户看的命令手册。

> **谁应该读这里**：在写一个 agent / pipeline / MCP 工具去调用 pdf-cli 的人。
>
> **想看每条命令的完整 flag 与示例**：参考 [skills/](../skills/) 目录（Claude Code skill 格式，对所有 LLM 都可读）或运行 `pdf-cli <cmd> --help`。
>
> **想要快速上手 demo**：参考根目录 [README.md](../README.md)。

## 这份文档与 skills/ 的关系

| 目录 | 受众 | 内容 |
|---|---|---|
| [skills/](../skills/) | Claude Code Skill 加载，也可被任意 LLM 读取 | 每个命令的 flag / 参数 / 输出 / 示例。命令式写法 |
| [document/](.) | 通用 agent 集成方 | skills/ 装不下的：错误码语义、异步链路、API 映射、端到端示例 |

两者**不重复**：skills 告诉你"命令怎么用"，document 告诉你"agent 应该怎么基于这些命令做决策"。

## 文件导航

| 文件 | 内容 | 必读优先级 |
|---|---|---|
| [proposal.md](proposal.md) | 顶层设计方案：目标、原则、关键决策、实现位置一览 | ★★★ 接手项目或想理解 why 时先读 |
| [conventions.md](conventions.md) | 退出码、JSON envelope、`--format` 语义、重试策略、命令鉴权要求 | ★★★ 集成前必读 |
| [errors.md](errors.md) | 每个退出码的来源、识别方式、agent 处理动作 | ★★★ |
| [async-flows.md](async-flows.md) | translate 与 tools 两条异步链路的状态机、轮询、终态判断 | ★★ |
| [api-mapping.md](api-mapping.md) | CLI 命令到后端 HTTP 接口的映射 | ★ 仅当需要绕过 CLI 直连后端时读 |
| [end-to-end.md](end-to-end.md) | 完整集成示例：免费翻译、会员翻译重试、tools 异步、流式文本、降级链 | ★★ 写代码时对照 |
| [full-cli-test-report.md](full-cli-test-report.md) | 历史测试报告，含已发现问题与修复记录 | 调试时参考 |
| [npm-publish.md](npm-publish.md) | 维护者发布流程：tag 命名、2FA / OTP、GitHub Release、常见错误排查 | 仅维护者发版时读 |

## 快速参考：命令分类

```
auth        登录、登出、状态
translate   PDF 翻译（含 free / arxiv / text 流式）
tools       PDF 工具集（merge / convert / split / 等）
user        个人资料、文件记录、API key
member      会员信息、权益、订单
other       公告、版本、帮助
```

异步命令（agent 决策最复杂的）：
- **translate 链路**：`upload → start/free → status → download`
- **tools 链路**：所有 `tools <action>` 高层命令默认同步等待；`tools job *` 是显式异步

详见 [async-flows.md](async-flows.md)。

## 集成检查清单（30 秒版）

写 agent 之前过一遍：

1. 总是带 `--format json`
2. 解析 stderr 的 JSON envelope，不要 grep 文本
3. 决策看 `exit_code` 和 `retryable`，不要看 message
4. 重试前确认命令幂等性（[conventions.md](conventions.md) §4.3）
5. 区分"命令失败"（exit 1–13, 20）和"任务失败"（exit 14）
6. `auth login` 由人工/会话初始化预先完成，agent 不自动登录
7. task-id / file-key / query-key 当 opaque 字符串处理

完整版见 [end-to-end.md](end-to-end.md) §7。

## 文档维护

- 修改 [internal/errors/errors.go](../internal/errors/errors.go) 的退出码常量时，同步更新 [conventions.md](conventions.md) §1 和 [errors.md](errors.md)
- 新增/修改命令的后端接口时，同步更新 [api-mapping.md](api-mapping.md)
- 新增异步命令时，同步更新 [async-flows.md](async-flows.md)
- 调整重试约定时，同步更新 [conventions.md](conventions.md) §4
