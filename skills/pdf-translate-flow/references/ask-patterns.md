# AskUserQuestion 措辞模板

每个决策节点的问题文本和选项格式。所有选项都用中文，简洁不啰嗦。

如果 `AskUserQuestion` 工具尚未加载，先调用：

```text
ToolSearch(query="select:AskUserQuestion", max_results=1)
```

---

## N3a · 免费还是高级（已登录）

```json
{
  "questions": [
    {
      "question": "用免费翻译还是高级翻译？",
      "header": "翻译类型",
      "multiSelect": false,
      "options": [
        {
          "label": "免费翻译",
          "description": "使用默认普通引擎，不消耗会员字符"
        },
        {
          "label": "高级翻译",
          "description": "可选普通或高级引擎、OCR，普通用户 2次/天·600字/次，会员消耗字符"
        }
      ]
    }
  ]
}
```

---

## N3b · 未登录-是否使用高级翻译

```json
{
  "questions": [
    {
      "question": "用免费翻译还是高级翻译？高级翻译需要先登录账号。",
      "header": "翻译类型",
      "multiSelect": false,
      "options": [
        {
          "label": "免费翻译（游客可用）",
          "description": "译文只在控制台打印，不下载到本地"
        },
        {
          "label": "高级翻译（需登录）",
          "description": "选此项我会给你登录命令并暂停流程"
        }
      ]
    }
  ]
}
```

---

## N4 · 选择目标语言（两阶段）

> **工具硬限制**：`AskUserQuestion` 每个 question 最多 4 个 options（schema 强制）。
> **不依赖工具自动 Other**：在实际 UI 里 Other 入口可能不可见，所以**显式**把"更多语言/手动输入"放进 options，占用 1 个槽位，确保用户一定看到下一步入口。

### N4-S1（屏 1：3 个语言 + 显式"更多"）

`langList` 前 3 项 + 显式更多入口：

```json
{
  "questions": [
    {
      "question": "翻成什么语言？",
      "header": "目标语言",
      "multiSelect": false,
      "options": [
        {"label": "简体中文 (zh-CN)","description": "中文（大陆/简体）"},
        {"label": "英语 (en)","description": "English"},
        {"label": "日语 (ja)","description": "日本語"},
        {"label": "更多语言…","description": "看其他常用语言或手动输入语言代码"}
      ]
    }
  ]
}
```

**用户选"更多语言…"** → 进入 N4-S2。

### N4-S2（屏 2：3 个语言 + 显式"手动输入"）

`langList` 第 4–6 项 + 显式手动输入入口：

```json
{
  "questions": [
    {
      "question": "选择目标语言（继续）",
      "header": "更多语言",
      "multiSelect": false,
      "options": [
        {"label": "韩语 (ko)",         "description": "한국어"},
        {"label": "繁体中文 (zh-TW)",  "description": "中文繁體"},
        {"label": "俄语 (ru)",         "description": "Русский"},
        {"label": "手动输入语言代码",   "description": "上面没有想要的语言时选这个"}
      ]
    }
  ]
}
```

**用户选"手动输入语言代码"** → agent 改用文本提示（**不再开 AskUserQuestion**），输出：

```text
没有命中常用语言。请直接告诉我你要的语言代码或语言名称，例如：
  - 翻成阿拉伯语
  - 用语言代码 ar

如果不确定支持哪些，请先运行：
  pdf-cli translate languages
查看完整列表，再告诉我代码即可。
```

然后**等待用户在对话里以自由文本回复**（不再调 AskUserQuestion）。

### N4 输入验证（用户文本回复后）

1. 拉 `pdf-cli translate languages --format json` 拿 `langList`
2. 把用户输入按下列顺序匹配：
   - 精确匹配 `code`（如 `ar`、`itb`）
   - 模糊包含匹配 `name`（如"阿拉伯"匹配"阿拉伯语"）
3. 命中 → 用对应 `code` 进入 upload；stdout 打印一行 `→ 目标语言: <name> (<code>)` 让用户确认
4. **未命中** → 不要瞎猜，直接打印：

   ```text
   你输入的「<input>」不在支持的语言列表里。请先运行：
     pdf-cli translate languages
   查看完整列表，找到对应代码后再告诉我。
   ```

   然后**终止本次流程**——让用户重新发起，不要循环追问。

### 跳过 N4 的条件

用户原话已显式指定语言（如"翻成日文"/"翻成 ar"）→ 直接做"输入验证"，跳过 S1/S2。

---

## N5 · 选择翻译引擎（两阶段）

> **工具硬限制**：每个 question 最多 4 个 options。所以采用两阶段：先选类型，再选具体引擎。

### N5-S1（屏 1：选引擎类型）

> **始终展示，不因会员态跳过**。会员同样可以选普通引擎（成本更低、倍率小），让用户主动选择。

```json
{
  "questions": [
    {
      "question": "先选引擎类型",
      "header": "引擎类型",
      "multiSelect": false,
      "options": [
        {
          "label": "普通引擎",
          "description": "成本低、倍率小（多为 0-1x），适合普通文档；所有登录用户都能用"
        },
        {
          "label": "高级引擎",
          "description": "更强模型（GPT/Claude/Gemini 等），倍率较高，仅会员可用"
        }
      ]
    }
  ]
}
```

**会员**（`vipLevel > 0`）在打开此问题前 stdout 提示一行：

```text
你是会员，两组引擎都可用。普通引擎倍率小消耗少，高级引擎质量更高。
```

**非会员**（`vipLevel == 0`）在打开此问题前 stdout 提示一行：

```text
你不是会员（vipLevel=0）。建议选普通引擎；选了高级引擎会被后端 quota 拦截。
普通用户限额：每日 2 次 / 每次最多 600 字符。
```

### N5-S2（屏 2：3 个引擎 + `other`）

根据 S1 选择，从 `engines` 拉过滤后的列表：

- S1=普通：过滤 `showFlag==1 && highLevelFlag==0`
- S1=高级：过滤 `showFlag==1 && highLevelFlag==1`

按 `sort` 升序取**前 3 个**作为引擎 options，第 4 个槽位放显式 `other`。label 含倍率，description 冗余 `engineName`。

```json
{
  "questions": [
    {
      "question": "选择具体引擎（倍率影响会员字符消耗）",
      "header": "翻译引擎",
      "multiSelect": false,
      "options": [
        {"label": "chatgpt-4omini · 0x",   "description": "engineName=chatgpt-4omini"},
        {"label": "gemini-1.5-pro · 1x",   "description": "engineName=gemini-1.5-pro"},
        {"label": "glm-4.5-flash · 1x",    "description": "engineName=glm-4.5-flash"},
        {"label": "other",                 "description": "手动输入引擎名称或先查询完整引擎列表"}
      ]
    }
  ]
}
```

（高级组示例：`chatgpt-4.1-mini · 1x` / `gemini-2.0-flash · 1x` / `deepseek-V3 · 1x` + `other`）

**用户选 `other`** → agent 改用文本提示，不再开 AskUserQuestion：

```text
没有命中想要的引擎。请直接告诉我引擎名称，例如：
  - 用 chatgpt-4.1
  - 引擎 deepseek-v3.1

也可以先运行：
  pdf-cli translate engines
查看完整列表（含倍率），再告诉我 engineName。
```

等待用户文本回复。

### N5 输入验证（用户文本回复后）

1. 拉 `pdf-cli translate engines --format json`
2. 过滤 `showFlag==1`
3. 把用户输入按下列顺序匹配：
   - 精确匹配 `engineName`（如 `chatgpt-4.1`）
   - 不区分大小写包含匹配 `engineShowName`
4. 命中 → 检查 `highLevelFlag` 是否与 S1 选的类型一致：
   - 一致 → 用 `engineName` 传给 `--engine`，stdout 打印 `→ 引擎: <showName> · <ratio>x`
   - 不一致 → stdout 提示 `你输入的引擎是<普通/高级>，与你刚才选的<高级/普通>类型不符。我按引擎实际类型走` 然后继续（**信号最强的是 engineName 匹配，类型可以自动纠正**）
5. **未命中** → 打印：

   ```text
   你输入的「<input>」不在支持的引擎列表里。请先运行：
     pdf-cli translate engines
   查看完整列表，找到 engineName 后再告诉我。
   ```

   然后**终止本次流程**。

**反推 engineName**：用户从 S2 选中卡片选项时，从 description 里解析 `engineName=<x>`。

### 跳过 N5 的条件

用户原话已显式指定引擎名（如"用 google 翻译"/"用 chatgpt-4.1"）→ 直接走"输入验证"，跳过 S1/S2。

---

## N6 · 是否启用 OCR

```json
{
  "questions": [
    {
      "question": "是否启用 OCR？扫描件 / 图片型 PDF 需要开启。",
      "header": "OCR",
      "multiSelect": false,
      "options": [
        {
          "label": "否（默认）",
          "description": "正常文本型 PDF，速度快"
        },
        {
          "label": "是（OCR）",
          "description": "扫描件、图片型 PDF，识别文字后再翻译，较慢"
        }
      ]
    }
  ]
}
```

---

## N8a · 是否下载译文

```json
{
  "questions": [
    {
      "question": "翻译完成。是否现在下载到本地？",
      "header": "下载译文",
      "multiSelect": false,
      "options": [
        {
          "label": "下载",
          "description": "保存到当前目录"
        },
        {
          "label": "稍后",
          "description": "记住 task-id，以后用 download 命令"
        }
      ]
    }
  ]
}
```

---

## 多个独立决策的合并问

> **工具限制**：每次 AskUserQuestion 最多 1–4 个 question。N4/N5 各自拆成两阶段后（S1 + S2），完整高级路径理论需要 5 个 question 槽位（N4-S1 / N4-S2 / N5-S1 / N5-S2 / N6），**超过上限**。所以**不能**一次性合并。

### 推荐策略

**首屏合并 3 个**（节省一来一回）：

```json
{
  "questions": [
    {"question": "翻成什么语言？", "header": "目标语言", ...},  // N4-S1
    {"question": "选择引擎类型", "header": "引擎类型", ...},  // N5-S1
    {"question": "是否启用 OCR？", "header": "OCR", ...}      // N6
  ]
}
```

然后根据用户在首屏的回答**按需触发**第二屏：

- N4-S1 选 Other → 第二轮加 N4-S2
- N5-S1 选完后 → 第二轮加 N5-S2
- 两个都触发 → 第二轮 2 个 question；只触发一个 → 第二轮 1 个 question

**最坏情况**：两次 AskUserQuestion 调用（首屏 3 + 二屏 ≤2），仍比纯顺序问（最多 5 次）省。

### 不要这么做

```json
// ❌ 5 个 question — schema 校验直接失败
{ "questions": [N4-S1, N4-S2, N5-S1, N5-S2, N6] }
```

---

## 取消处理

`AskUserQuestion` 用户取消时（返回空或 cancel 信号）：

- 立即停止流程
- 打印 `已取消。如需重试请重新发起翻译。`
- 不要清理任何状态（file-key 由后端 TTL 处理）
