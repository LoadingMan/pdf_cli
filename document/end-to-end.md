# end-to-end

完整的 agent 集成示例。每个示例展示一个最小可运行的 agent 决策循环，包含错误处理、重试、状态判断。代码片段是 **shell + jq 伪代码**，方便任意 agent 实现移植——不绑定 LangChain / Claude SDK / OpenAI tool-use 任何特定框架。

> **前提**：所有示例假设 `pdf-cli` 在 PATH 中、`jq` 可用、`--format json` 已开启。Agent 实现应在自己一侧把这些步骤翻译成对应语言。

## 1. 免费翻译一篇 PDF（无登录）

最简场景。无需账号、无需额度、自动降级。

```bash
#!/bin/bash
set -e

INPUT="./paper.pdf"
TARGET_LANG="zh-CN"
OUTPUT="./paper.zh.pdf"

# 1. 上传（未登录会自动走 free 路径）
upload_json=$(pdf-cli --format json translate upload --file "$INPUT" --free)
file_key=$(echo "$upload_json" | jq -r '.tmpFileName // .key')

# 2. 发起免费翻译
start_json=$(pdf-cli --format json translate free \
  --file-key "$file_key" --to "$TARGET_LANG")
task_id=$(echo "$start_json" | jq -r '.blobFileName | sub("\\..*$"; "")')

# 3. 阻塞等待终态。退出码即结局。
status_err=$(pdf-cli --format json translate status --task-id "$task_id" --wait 2>&1 1>/dev/null)
status_exit=$?
if [ $status_exit -ne 0 ]; then
  case $status_exit in
    14)
      reason=$(echo "$status_err" | jq -r '.details.fail_reason // "unknown"')
      term=$(echo "$status_err" | jq -r '.details.status // "unknown"')
      echo "翻译失败 status=$term reason=$reason" >&2
      pdf-cli --format json translate history --page 1 --page-size 5 >&2
      exit 1
      ;;
    10|11|12|13)
      # 不应该出现在 status 命令上（除非 wait 模式下网络挂了）
      echo "暂时性错误 exit=$status_exit，建议人工介入" >&2
      exit 1
      ;;
    *)
      echo "未预期错误 exit=$status_exit" >&2
      exit 1
      ;;
  esac
fi

# 4. 下载
pdf-cli translate download --task-id "$task_id" -o "$OUTPUT"
echo "✓ 翻译完成: $OUTPUT"
```

**关键决策点**：
- 步骤 1 不检查登录状态，让 CLI 自动降级
- 步骤 3 用 `--wait` 把轮询交给 CLI，agent 只看退出码
- 步骤 4 不再检查 task 状态，因为步骤 3 已经退出 0 = done

## 2. 会员翻译，带重试与额度检查

适用于已登录账号、有 VIP 等级、可能命中限流。

```bash
#!/bin/bash

INPUT="./paper.pdf"
TARGET_LANG="zh-CN"
ENGINE="gpt-4o-mini"

# 0. 前置检查：登录状态 + 额度
auth_json=$(pdf-cli --format json auth status 2>&1) || {
  echo "未登录，请先 pdf-cli auth login" >&2
  exit 5
}

remain=$(echo "$auth_json" | jq -r '.remainScore // 0')
if [ "$remain" -lt 1 ]; then
  echo "额度不足（剩余 $remain），尝试免费路径" >&2
  exec ./free-translate.sh "$@"
fi

# 1. 上传（会员模式）
upload_out=$(pdf-cli --format json translate upload --file "$INPUT" 2>&1)
upload_exit=$?
if [ $upload_exit -ne 0 ]; then
  case $upload_exit in
    5) echo "token 失效，重新登录后重试" >&2; exit 5 ;;
    7) echo "额度耗尽，降级到 free" >&2; exec ./free-translate.sh "$@" ;;
    *) echo "$upload_out" >&2; exit $upload_exit ;;
  esac
fi
file_key=$(echo "$upload_out" | jq -r '.tmpFileName')

# 2. 发起翻译，带重试（针对 10/11/12/13）
attempt=0
delay=1
while true; do
  attempt=$((attempt+1))
  start_out=$(pdf-cli --format json translate start \
    --file-key "$file_key" --to "$TARGET_LANG" --engine "$ENGINE" 2>&1)
  start_exit=$?
  [ $start_exit -eq 0 ] && break

  case $start_exit in
    10|11|12|13)
      if [ $attempt -ge 3 ]; then
        echo "重试 3 次后仍失败 exit=$start_exit" >&2
        echo "$start_out" >&2
        exit $start_exit
      fi
      sleep $delay
      delay=$((delay*2))
      ;;
    7)
      echo "额度在中途耗尽，降级到 free" >&2
      exec ./free-translate.sh "$@"
      ;;
    *)
      echo "$start_out" >&2
      exit $start_exit
      ;;
  esac
done
task_id=$(echo "$start_out" | jq -r '.blobFileName | sub("\\..*$"; "")')

# 3. 等待终态
pdf-cli --format json translate status --task-id "$task_id" --wait > /dev/null
status_exit=$?
[ $status_exit -ne 0 ] && exit $status_exit

# 4. 下载
pdf-cli translate download --task-id "$task_id" -o "./out.pdf"
```

**关键决策点**：
- 步骤 0 在动作前检查 `remainScore` 避免发起后才发现额度不足
- 步骤 1 的 exit 7 触发降级而不是失败
- 步骤 2 只对 retryable 集合做重试，其他错误立即放弃
- 步骤 3 不重试，因为 status --wait 内部已经在轮询，外层重试无意义

## 3. PDF 合并，处理任务超时

`tools merge` 是同步命令，但内部走异步链路。可能在 240s 后超时退出 12。超时时 stderr envelope 的 `details.query_key` 字段会带上当前 query-key，agent 可以无缝切换到异步轮询。

```bash
#!/bin/bash

merge_out=$(pdf-cli --format json tools merge \
  --files a.pdf,b.pdf,c.pdf -o merged.pdf 2>&1)
merge_exit=$?

case $merge_exit in
  0)
    echo "✓ 合并完成"
    ;;
  12)
    # 超时——从 details.query_key 拿到任务标识，自己继续轮询
    query_key=$(echo "$merge_out" | jq -r '.details.query_key // empty')
    if [ -z "$query_key" ]; then
      echo "超时但 envelope 未带 query_key" >&2
      exit 12
    fi
    echo "任务仍在跑，切换到异步轮询: $query_key"
    while true; do
      status_out=$(pdf-cli --format json tools job status --query-key "$query_key" 2>&1)
      status_exit=$?
      case $status_exit in
        14)
          echo "任务失败: $(echo "$status_out" | jq -r '.message')" >&2
          exit 14
          ;;
        0)
          state=$(echo "$status_out" | jq -r '.state')
          case "$state" in
            SUCCESS)
              pdf-cli tools job download --query-key "$query_key" -o merged.pdf
              break
              ;;
            PROCESSING|PENDING|RECEIVED)
              sleep 5
              ;;
            *)
              echo "未知状态: $state" >&2
              exit 1
              ;;
          esac
          ;;
        *)
          echo "查询失败 exit=$status_exit" >&2
          exit $status_exit
          ;;
      esac
    done
    ;;
  14)
    # 任务失败，envelope 带 query_key 方便后续诊断
    query_key=$(echo "$merge_out" | jq -r '.details.query_key // empty')
    echo "任务失败: $(echo "$merge_out" | jq -r '.message')" >&2
    [ -n "$query_key" ] && echo "query_key: $query_key" >&2
    exit 14
    ;;
  *)
    echo "$merge_out" >&2
    exit $merge_exit
  ;;
esac
```

**details 字段的好处**：之前需要 grep `message` 拿 query-key，依赖文案稳定性；现在直接 `jq '.details.query_key'`，agent 不再受 i18n 或文案改动影响。

## 4. 文本翻译流式

文本翻译是 SSE 流，CLI 把流的每个 chunk 直接打到 stdout。

```bash
# 阻塞模式：一次性拿完整翻译
result=$(pdf-cli --format json translate text \
  --text "Hello, world." --to zh-CN --engine gpt-4o-mini 2>&1)
exit_code=$?

case $exit_code in
  0)
    echo "$result" | jq -r '.text // .'
    ;;
  14)
    # SSE 流中收到了 [ERROR] 事件
    echo "翻译失败: $(echo "$result" | jq -r '.message')" >&2
    exit 14
    ;;
  *)
    echo "$result" >&2
    exit $exit_code
    ;;
esac
```

如果 agent 想要流式渲染（边收边显示），不要用 `--format json`，直接 pipe pretty 模式的 stdout：

```bash
pdf-cli translate text --text "..." --to zh-CN | while IFS= read -r chunk; do
  printf '%s' "$chunk"  # 实时显示
done
```

## 5. 完整的鉴权降级链

agent 经常需要决策"先试 VIP 路径，失败后降级到免费路径"。下面是一个通用的决策函数。

```bash
translate_pdf() {
  local input="$1" target="$2" output="$3"

  # 试一：会员路径
  if pdf-cli --format json auth status > /dev/null 2>&1; then
    if try_vip_path "$input" "$target" "$output"; then
      return 0
    fi
    local rc=$?
    # 6 (permission), 7 (quota) 才降级；其他错误直接报
    if [ $rc -ne 6 ] && [ $rc -ne 7 ]; then
      return $rc
    fi
    echo "VIP 路径不可用 (exit $rc)，降级到免费路径" >&2
  fi

  # 试二：免费路径
  try_free_path "$input" "$target" "$output"
}
```

## 6. 决策表：从退出码到下一步动作

| exit | 立即动作 | 长期处理 |
|---|---|---|
| 0 | 解析 stdout，进入下一步 | – |
| 2 | 修命令行 | 反馈给生成 prompt 的模型 |
| 3 | 改参数 | 检查上一步输出解析逻辑 |
| 4 | 提示人工介入 | 检查 ~/.config/pdf-cli/ 权限 |
| 5 | 提示 `auth login` | 集成场景应在会话开始前预登录 |
| 6 | 降级到 free 路径或换号 | 提示用户升级 |
| 7 | 降级到 free 路径 | 提示用户充值 |
| 8 | 用 history 验证 ID 后决定 | 检查 ID 来源 |
| 9 | 调 status 看真实状态 | 改用查询而不是变更命令 |
| 10 | 退避 5s ×2 重试，最多 5 次 | 串行化并发调用 |
| 11 | 立即重试 1 次，再退避 | 检查网络环境 |
| 12 | 重试 + 增大超时 | 对非幂等命令先 status 确认 |
| 13 | 退避 2s ×2 重试，最多 3 次 | 持续 5xx 上报后端 |
| 14 | 读 message，决定整链路重试 | 不要重试 status 本身 |
| 1, 20 | 上报、停止 | 收集 stderr 作为 bug |

## 7. agent 实现的最小检查清单

- [ ] 总是带 `--format json`
- [ ] 解析 stderr 的 JSON envelope，而不是 grep `Error:` 行
- [ ] 决策只看 `exit_code` / `type` / `retryable`，不要 grep `message`
- [ ] 重试前判断命令幂等性（见 [conventions.md](conventions.md) §4.3）
- [ ] 重试上限明确（建议 3 次），防止无限循环
- [ ] 区分"任务失败"（exit 14）和"命令失败"（exit 1–13, 20）
- [ ] 对非幂等命令，超时（exit 12）后先调 status/history 再决定是否重试
- [ ] `auth login` 不要让 agent 自动调（需要密码交互）
- [ ] task-id / file-key / query-key 当作 opaque 字符串，不要解析其内部结构
