# 多轮对话解决方案设计

## 目标
为现有的 `OpsAgent` 系统添加多轮对话能力，使其能够记住用户的上下文并在多轮交互中提供更准确的响应。

---

## 设计方案

### 1. **会话上下文管理**
- **存储历史记录**：为每个用户会话维护一份对话历史记录。
- **存储方式**：
  - **内存存储**：适合短期会话，使用 `map` 或 `session` 存储用户的对话历史。
  - **持久化存储**：适合长期会话，使用 Redis 或数据库（如 MongoDB）存储历史记录。

---

### 2. **API 接口设计**
- **请求结构**：
  - 增加 `history` 字段，用于传递用户的对话历史。
  - 示例：
    ```json
    {
      "instructions": "请分析这个Pod的状态",
      "args": "",
      "history": [
        "用户：请检查这个Pod的日志",
        "助手：日志显示Pod运行正常，没有错误信息"
      ]
    }
    ```

- **响应结构**：
  - 返回更新后的 `history`，以便客户端维护上下文。
  - 示例：
    ```json
    {
      "response": "Pod状态正常，没有发现异常。",
      "history": [
        "用户：请检查这个Pod的日志",
        "助手：日志显示Pod运行正常，没有错误信息",
        "用户：请分析这个Pod的状态",
        "助手：Pod状态正常，没有发现异常。"
      ]
    }
    ```

---

### 3. **对话历史的构建**
- **方法**：将用户输入和助手响应按顺序追加到 `history` 中。
- **格式**：
  - 每条记录包括角色（用户或助手）和内容。
  - 示例：
    ```json
    [
      { "role": "user", "content": "请检查这个Pod的日志" },
      { "role": "assistant", "content": "日志显示Pod运行正常，没有错误信息" }
    ]
    ```

---

### 4. **与 LLM 的集成**
- **输入历史记录**：将 `history` 转换为 OpenAI API 所需的 `ChatCompletionMessage` 格式。
- **示例代码**：
  ```go
  func buildPrompt(history []string) []openai.ChatCompletionMessage {
      var messages []openai.ChatCompletionMessage
      for _, h := range history {
          messages = append(messages, openai.ChatCompletionMessage{
              Role:    "user",
              Content: h,
          })
      }
      return messages
  }
  ```

---

### 5. **上下文截断**
- **原因**：LLM 对输入的 token 数量有限制（如 GPT-4 的限制为 8192 tokens）。
- **解决方案**：
  - 优先保留最近的对话记录。
  - 使用 `llms.ConstrictMessages` 方法对历史记录进行截断。

### 5. **上下文优化策略**
- **滑动窗口机制**：
  ```go
  const MaxTurns = 10 // 保留最近的10轮对话
  
  type HistoryManager struct {
      maxTurns int
      history  []Message
  }
  ```

- **Token 数量控制**：
  - 设置最大 token 限制（如 4000 tokens）
  - 从最新消息开始保留，直到达到限制

- **会话摘要机制**：
  ```json
  {
    "summary": "用户之前询问了Pod的日志状态，确认运行正常",
    "recent_history": [
      { "role": "user", "content": "请分析这个Pod的状态" },
      { "role": "assistant", "content": "Pod状态正常" }
    ]
  }
  ```

- **相关性过滤**：
  - 使用向量相似度匹配最相关的历史记录
  - 仅保留相似度大于阈值（如0.7）的历史消息

### 实现优先级：
1. 实现基础的滑动窗口机制
2. 添加 token 计数控制
3. 集成会话摘要功能
4. 添加相关性过滤（可选）

### 存储策略：
- **活跃会话**：Redis (TTL: 30分钟)
- **历史会话**：向量数据库
  ```go
  type Session struct {
      ID        string    `json:"id"`
      Summary   string    `json:"summary"`
      History   []Message `json:"history"`
      UpdatedAt time.Time `json:"updated_at"`
  }
  ```

---

### 6. **性能优化**
- **缓存**：对频繁访问的会话历史进行缓存（如使用 Redis）。
- **异步存储**：将历史记录的持久化存储操作放到异步任务中，减少接口响应时间。

---

### 7. **错误处理**
- **历史记录为空**：如果 `history` 为空，助手应根据 `instructions` 提供初始响应。
- **历史记录过长**：截断历史记录并记录警告日志。

---

### 8. **安全性**
- **敏感信息过滤**：在存储和传递历史记录时，过滤掉可能包含敏感信息的内容。
- **加密存储**：对持久化存储的历史记录进行加密。

---

## 实现步骤

### 1. **修改请求结构**
- 在 `ExecuteRequest` 中添加 `History` 字段。

### 2. **更新处理逻辑**
- 在 `Execute` 方法中：
  - 将用户输入追加到 `history`。
  - 调用 `buildPrompt` 方法生成对话上下文。
  - 将更新后的 `history` 返回给客户端。

### 3. **优化助手逻辑**
- 在 `AssistantWithConfig` 方法中：
  - 接收完整的对话历史。
  - 将历史记录传递给 LLM。

### 4. **测试**
- 编写单元测试和集成测试，验证多轮对话功能的正确性。

---

## 示例流程

### 请求
```json
{
  "instructions": "请分析这个Pod的状态",
  "args": "",
  "history": [
    "用户：请检查这个Pod的日志",
    "助手：日志显示Pod运行正常，没有错误信息"
  ]
}
```

### 响应
```json
{
  "response": "Pod状态正常，没有发现异常。",
  "history": [
    "用户：请检查这个Pod的日志",
    "助手：日志显示Pod运行正常，没有错误信息",
    "用户：请分析这个Pod的状态",
    "助手：Pod状态正常，没有发现异常。"
  ]
}
```

---

## 总结
通过上述设计，`OpsAgent` 系统可以支持多轮对话功能，提升用户体验和交互效率。