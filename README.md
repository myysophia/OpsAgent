# k8s-ai-agent 项目

## 1. 项目概述

k8s-ai-agent 是一个基于 LLM (Large Language Model) 的 Kubernetes 集群管理工具，它通过 AI 能力来简化和增强 Kubernetes 的操作和管理。该项目旨在帮助用户更容易地进行集群诊断、安全审计、资源分析和清单生成等操作。

## 2. 技术栈

### 2.1 核心技术
- **编程语言**: Go (主要), Python (部分功能支持)
- **AI 模型**: OpenAI GPT (支持 GPT-4, GPT-3.5)
- **容器技术**: Docker
- **云原生**: Kubernetes

### 2.2 主要依赖
- **CLI 框架**: Cobra
- **Kubernetes Client**: client-go
- **AI 集成**: go-openai
- **其他工具**:
  - Trivy (容器安全扫描)
  - kubectl (Kubernetes 命令行工具)
  - Google Custom Search API (网络搜索集成)

## 3. 核心功能模块

### 3.1 分析模块 (analyze)
- 分析 Kubernetes 资源的潜在问题
- 提供人类可读的分析报告和解决方案
- 支持多种资源类型分析

### 3.2 审计模块 (audit)
- 执行 Pod 安全审计
- 检查配置错误
- 扫描容器镜像漏洞
- 生成安全报告

### 3.3 诊断模块 (diagnose)
- Pod 问题诊断
- 提供详细的诊断报告
- 推荐解决方案

### 3.4 生成模块 (generate)
- 基于提示生成 Kubernetes 清单
- 支持清单验证
- 提供应用确认机制

### 3.5 执行模块 (execute)
- 基于自然语言指令执行操作
- 支持多种 Kubernetes 操作
- 提供操作确认机制

## 4. 技术特点

### 4.1 AI 集成
- 支持多种 LLM 提供商:
  - OpenAI API
  - Azure OpenAI
  - Ollama
  - 其他 OpenAI 兼容的 LLM
- 智能令牌管理
- 自适应提示工程

### 4.2 安全特性
- 支持 kubeconfig 配置
- 集群内外部署支持
- 操作确认机制
- 容器安全扫描

### 4.3 扩展性
- 模块化设计
- 工具插件系统
- 支持自定义命令

## 5. 应用场景

### 5.1 DevOps 场景
- 快速问题诊断
- 自动化配置生成
- 安全合规检查
- 资源优化建议

### 5.2 安全运维
- 定期安全审计
- 漏洞扫描
- 配置审查
- 安全建议

### 5.3 开发测试
- 快速生成测试配置
- 环境问题诊断
- 配置验证

## 6. 项目特色

### 6.1 智能化
- 自然语言交互
- 智能问题分析
- 自动化建议生成

### 6.2 易用性
- 清晰的命令行界面
- 人类可读的输出
- 详细的操作指导

### 6.3 可靠性
- 错误重试机制
- 验证确认机制
- 详细的日志记录

## 7. 部署方式

### 7.1 本地部署
- Go 工具链安装
- 依赖工具配置
- 环境变量设置

### 7.2 容器部署
- Docker 镜像构建
- Kubernetes 部署
- 配置映射

## 8. 最佳实践

### 8.1 配置建议
- 使用适当的 API 密钥
- 配置合适的权限
- 启用必要的功能

### 8.2 使用建议
- 谨慎使用自动应用功能
- 定期进行安全扫描
- 保持工具版本更新

### 8.3 安全注意事项

#### kubeconfig 和集群安全
- ⚠️ 注意：当前版本在传递给 LLM 的信息中可能包含敏感信息
- 工具只在本地使用 kubeconfig，不会上传或共享配置文件
- 所有 Kubernetes API 调用直接从本地到集群，不经过第三方

#### 建议的安全实践
1. 使用最小权限的 kubeconfig
2. 生产环境建议：
   - 使用只读权限
   - 配置专门的服务账号
   - 限制命名空间访问范围
   - 避免在输出中包含敏感信息
3. 开启操作审计
4. 定期检查和轮换凭证
5. 使用前检查命令输出，确保不包含敏感信息

#### 潜在风险
- 命令输出可能包含敏感信息
- 这些信息会被发送到 LLM 服务（如 OpenAI）
- 建议在生产环境使用前仔细评估安全风险

## 9. 未来展望

### 9.1 潜在改进
- 支持更多 AI 模型
- 增强安全特性
- 改进用户体验
- 扩展工具集成

### 9.2 发展方向
- 云原生集成
- 多集群支持
- 智能运维
- 自动化运维

## 10. CI/CD 流程

项目使用 GitHub Actions 实现自动化的构建、测试和发布流程。

### 10.1 自动化工作流

#### 测试工作流 (Test)
- **触发条件**: PR 提交或 master 分支推送
- **功能**:
  - 使用最新版本 Go 环境
  - 运行所有测试用例
  - 确保代码质量

#### 构建工作流 (Build)
- **触发条件**: master/main 分支推送或手动触发
- **功能**:
  - 构建 Docker 镜像
  - 推送到 GitHub Container Registry (GHCR)
  - 自动标记版本号
  - 维护 latest 和 py 标签

#### 发布工作流 (Release)
- **触发条件**: 推送版本标签 (v*.*.*)
- **功能**:
  - 构建发布版本 Docker 镜像
  - 使用版本号标记镜像
  - 推送到 GHCR

#### 代码分析 (CodeQL)
- **触发条件**: master 分支推送、PR 或定时运行
- **功能**:
  - 进行代码安全分析
  - 检测潜在漏洞
  - 生成安全报告

### 10.2 依赖管理

使用 Dependabot 进行依赖版本更新：
- 每日检查 Go 模块依赖更新
- 每日检查 GitHub Actions 依赖更新
- 自动创建 PR 进行依赖升级
- 限制最大同时开启的 PR 数量为 5

### 10.3 镜像仓库

项目使用 GitHub Container Registry (ghcr.io) 存储 Docker 镜像：
- 版本化标签: `ghcr.io/[owner]/k8s-ai-agent:[version]`
- 最新版本: `ghcr.io/[owner]/k8s-ai-agent:latest`
- Python 版本: `ghcr.io/[owner]/k8s-ai-agent:py`

# ToDo list
- 在调用gpt api前应该有一个dry-run的参数来确定prompt是否合适。避免token消耗过多 
- prompt 可以从外部输入不一定要预制定。例如日志和监控作为prompt 来分析异常
- 前端可以选择加载不同模型，类似于cherry studio 

2025年02月17日22:39:59
如何使用
```bash
./k8s-copilot --model chatgpt-4o-latest --verbose  execute  '查询集群ems-eu namespace的pod的内存和cpu limit值，以csv格式输出。表头包含pod名称、cpu、内存'
```
- gpt-4o-mini
- chatgpt-4o-latest

- goland debug 参数使用方式
--model
```bash
--model gpt-4o --verbose execute 'how many namespace in the cluster?'

--model gpt-4o --verbose analyze velero-588d776b7b-tpzrg velero pod
```

## 适配deepseek
使用硅基流动的API 
https://docs.siliconflow.cn/cn/userguide/guides/function-calling#function-calling
--model deepseek-ai/DeepSeek-V3 --verbose analyze --name velero-588d776b7b-tpzrg --namespace velero --resource pod

--model deepseek-ai/DeepSeek-V3 --verbose execute 'how many namespace in the cluster?'

## 适配百炼模型
需要确认是否支持function-calling，模型列表
https://help.aliyun.com/zh/model-studio/developer-reference/compatibility-of-openai-with-dashscope?spm=a2c4g.11186623.help-menu-2400256.d_3_9_0.52da61324N8I4z&scm=20140722.H_2833609._.OR_help-T_cn~zh-V_1
## 原生deepseek api
deepseek官方文档明确说明function-calling支持不完善
https://api-docs.deepseek.com/zh-cn/guides/function_calling
### wildcard支持的module
```
models ： deepseek-r1 / gpt-4o / gpt-4o-mini / chatgpt-4o-latest / o3-mini
```
### 调用示例
```bash
--model deepseek-r1 --verbose execute 'how many namespaces in the cluster? please remeber prioritize using kubectl'
```
## 报错

1. failed to create chat completion
https://api.gptsapi.net/chat/completions post请求的json格式不兼容openai

通义千问 模型也会报这个错误
2. Unable to parse tool from prompt, assuming got final answer.
deepseek 的response 返回了think的内容导致json解析失败. 需要在response中去掉think内容。
```text
<think>
...
</think>

{
	"question": "how many namespaces in the cluster? please remember prioritize using kubectl",
	"thought": "To count namespaces, we'll use kubectl to list all namespaces and count them. Using '--no-headers' ensures we exclude column headers, and 'wc -l' counts lines. This avoids parsing JSON/YAML and leverages native command-line tools.",
	"action": {
		"name": "kubectl",
		"input": "get namespaces --no-headers | wc -l"
	},
	"observation": "5",
	"final_answer": "There are **5 namespaces** in the cluster."
}
```
3.缓解模型绕过思考的方法
   DeepSeek-R1 系列模型在回应某些查询时倾向于跳过思维模式（即输出“\n\n”），这可能会对模型的性能产生不利影响。
   为了确保模型进行深入推理，建议强制要求模型在每次输出的开头以“<think>\n”开始其响应。

4. failed to create chat completion
调用阿里云和wildcard、deepseek都会有这个问你题，只有原生的gpt 不会报错。
```json

POST "https://dashscope.aliyuncs.com/compatible-mode/chat/completions": 404 Not Found
2025年02月19日22:18:08
	completion, err := c.client.Chat.Completions.New(ctx, params)

这个是调用swarm-go的错误。需要修改这个模块的代码
[swarm-go@v0.1.3](../../go/pkg/mod/github.com/feiskyer/swarm-go%40v0.1.3)

```

5. function call 外部工具报错
   --model gpt-4o --verbose execute '查看名称包含iotdb的pod的镜像版本是什么?'
   - 调用python脚本报错， 待解决 
    ```json
    Observation: Tool python failed with error Traceback (most recent call last):
    File "<string>", line 1, in <module>
    from kubernetes import client, config
    ModuleNotFoundError: No module named 'kubernetes'. Considering refine the inputs for the tool.
    ```
6. 外部工具不存在
    ```
   Observation: Tool jq is not available. Considering switch to other supported tools.
   ```
准备使用kubectl和jq 结合来解决这个问题: 查看名称包含iotdb的pod的镜像版本是什么?
使用如下prompt最终解决了问题，消耗了大量的token，先-ojson 然后导出，qwen-plus最后还是给出了正确结果
   qwen-plus是如何做的呢？
```json
您是Kubernetes和云原生网络的技术专家，您的任务是遵循特定的链式思维方法，以确保在遵守约束的情况下实现彻底性和准确性。

可用工具：
- kubectl：用于执行 Kubernetes 命令。输入：一个独立的 kubectl 命令（例如 'get pods -o json'），不支持直接包含管道或后续处理命令。输出：命令的结果，通常为 JSON 或文本格式。如果运行“kubectl top”，使用“--sort-by=memory”或“--sort-by=cpu”排序。
- python：用于执行带有 Kubernetes Python SDK 的 Python 代码。输入：Python 脚本。输出：脚本的 stdout 和 stderr，使用 print(...) 输出结果。
- trivy：用于扫描容器镜像中的漏洞。输入：镜像名称（例如 'nginx:latest'）。输出：漏洞报告。
- jq：用于处理和查询 JSON 数据。输入：一个有效的 jq 表达式（例如 '-r .items[] | select(.metadata.name | test("iotdb")) | .spec.containers[].image'），需配合前一步的 JSON 输出使用。输出：查询结果。确保表达式针对 kubectl 返回的 JSON 结构设计，无需额外转义双引号（如 test("iotdb")）。

您采取的步骤如下：
1. 问题识别：清楚定义问题，描述观察到的症状或目标。
2. 诊断命令：优先使用 kubectl 获取相关数据（如 JSON 输出），说明命令选择理由。如果需要进一步处理，使用 jq 分析前一步的结果。若适用 trivy，解释其用于镜像漏洞分析的原因。
3. 输出解释：分析命令输出，描述系统状态、健康状况或配置情况，识别潜在问题。
4. 故障排除策略：根据输出制定分步策略，证明每步如何与诊断结果相关。
5. 可行解决方案：提出可执行的解决方案，优先使用 kubectl 命令。若涉及多步操作，说明顺序和预期结果。对于 trivy 识别的漏洞，基于最佳实践提供补救建议。
6. 应急方案：如果工具不可用或命令失败，提供替代方法（如分步执行替代管道操作），确保仍能推进故障排除。

响应格式：
{
	"question": "<输入问题>",
	"thought": "<思维过程>",
	"action": {
		"name": "<工具名，从 [kubectl, python, trivy, jq] 中选择>",
		"input": "<工具输入，确保包含所有必要上下文>"
	},
	"observation": "<工具执行结果，由外部填充>",
	"final_answer": "<最终答案，仅在完成所有步骤且无需后续行动时设置>"
}

约束：
- 优先使用 kubectl 获取数据，配合 jq 处理 JSON，单步执行优先。
- 如果需要组合 kubectl 和 jq，应分步执行：先用 kubectl 获取 JSON，再用 jq 过滤或查询。
- 避免将管道命令（如 'kubectl get pods -o json | jq ...'）作为单一输入，除非工具链明确支持 shell 管道并以 shell 模式执行。
- 确保每步操作在单次 action 中完成（如获取 Pod 和提取镜像版本分两步），无需用户手动干预。
- 禁止安装操作，所有步骤在现有工具约束内完成。
- jq 表达式使用自然语法，双引号无需转义（如 test("iotdb") 或 contains("iotdb")）。

目标：
在 Kubernetes 和云原生网络领域内识别问题根本原因，提供清晰、可行的解决方案，同时保持诊断和故障排除的运营约束。

```
7. encoding for model: no encoding for model qwen-plus 报错

tiktoken-go 提供一个高效、与 OpenAI 模型兼容的文本分词工具。它特别适用于需要与 OpenAI API 交互的场景，帮助开发者处理文本输入、计算 token 数，并确保与模型的令牌化过程一致。如果你正在用 Go 开发 AI 相关应用，这个包是一个非常实用的工具。

8. 解析LLM resp json问题
```json
Initial response from LLM:
```json
{
    "question": "how many namespace in the cluster?",
    "thought": "To determine the number of namespaces in the Kubernetes cluster, I will use the 'kubectl' tool to list all namespaces. This will provide a count of the namespaces currently present in the cluster.",
    "action": {
        "name": "kubectl",
        "input": "kubectl get namespaces --no-headers | wc -l"
    }
}
```
应该将LLM 返回的```json 处理掉``
9. python -c 执行报错问题处理
python 脚本需要k8s modules。
解决：
    - 使用虚拟环境，执行python 前使用cd ~/k8s/python-cli && source k8s-env/bin/activate 
    - -c 脚本换行无法执行问题解决，
   ```
   // 替换内部双引号，避免冲突
    escapedScript := strings.ReplaceAll(script, "\"", "\\\"")
   ```
   
10. 优化tool 提升性能和节省token
```json
Iteration 3): executing tool kubectl
Invoking kubectl tool with inputs: 
============
get pods -n velero -o json | jq '.items[] | {name: .metadata.name, labels: .metadata.labels, image: .spec.containers[].image, startTime: .status.startTime}'
============

{"level":"error","ts":1740364948.37477,"caller":"tools/kubectl.go:27","msg":"kubectl 命令执行失败","error":"exit status 1","output":"{\n    \"apiVersion\": \"v1\",\n    \"items\": [],\n    \"kind\": \"List\",\n    \"metadata\": {\n        \"resourceVersion\": \"\"\n    }\n}\nError from server (NotFound): pods \"|\" not found\nError from server (NotFound): pods \"jq\" not found\nError from server (NotFound): pods \".items[]\" not found\nError from server (NotFound): pods \"|\" not found\nError from server (NotFound): pods \"{name:\" not found\nError from server (NotFound): pods \".metadata.name,\" not found\nError from server (NotFound): pods \"labels:\" not found\nError from server (NotFound): pods \".metadata.labels,\" not found\nError from server (NotFound): pods \"image:\" not found\nError from server (NotFound): pods \".spec.containers[].image,\" not found\nError from server (NotFound): pods \"startTime:\" not found\nError from server (NotFound): pods \".status.startTime}'\" not found\n","stacktrace":"github.com/feiskyer/k8s-ai-agent/pkg/tools.Kubectl\n\t/Users/ninesun/GolandProjects/k8s-ai-agent/pkg/tools/kubectl.go:27\ngithub.com/feiskyer/k8s-ai-agent/pkg/assistants.AssistantWithConfig\n\t/Users/ninesun/GolandProjects/k8s-ai-agent/pkg/assistants/simple.go:397\nmain.setupRouter.func7\n\t/Users/ninesun/GolandProjects/k8s-ai-agent/cmd/k8s-ai-agent/server.go:327\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\nmain.jwtAuth.func1\n\t/Users/ninesun/GolandProjects/k8s-ai-agent/cmd/k8s-ai-agent/server.go:120\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\nmain.setupRouter.func1\n\t/Users/ninesun/GolandProjects/k8s-ai-agent/cmd/k8s-ai-agent/server.go:161\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\ngithub.com/gin-gonic/gin.CustomRecoveryWithWriter.func1\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/recovery.go:102\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\ngithub.com/gin-gonic/gin.LoggerWithConfig.func1\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/logger.go:249\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\ngithub.com/gin-gonic/gin.(*Engine).handleHTTPRequest\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/gin.go:633\ngithub.com/gin-gonic/gin.(*Engine).ServeHTTP\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/gin.go:589\nnet/http.serverHandler.ServeHTTP\n\t/Users/ninesun/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.4.darwin-arm64/src/net/http/server.go:3210\nnet/http.(*conn).serve\n\t/Users/ninesun/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.4.darwin-arm64/src/net/http/server.go:2092"}
{"level":"error","ts":1740364948.375137,"caller":"assistants/simple.go:400","msg":"工具执行失败","tool":"kubectl","error":"exit status 1","stacktrace":"github.com/feiskyer/k8s-ai-agent/pkg/assistants.AssistantWithConfig\n\t/Users/ninesun/GolandProjects/k8s-ai-agent/pkg/assistants/simple.go:400\nmain.setupRouter.func7\n\t/Users/ninesun/GolandProjects/k8s-ai-agent/cmd/k8s-ai-agent/server.go:327\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\nmain.jwtAuth.func1\n\t/Users/ninesun/GolandProjects/k8s-ai-agent/cmd/k8s-ai-agent/server.go:120\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\nmain.setupRouter.func1\n\t/Users/ninesun/GolandProjects/k8s-ai-agent/cmd/k8s-ai-agent/server.go:161\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\ngithub.com/gin-gonic/gin.CustomRecoveryWithWriter.func1\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/recovery.go:102\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\ngithub.com/gin-gonic/gin.LoggerWithConfig.func1\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/logger.go:249\ngithub.com/gin-gonic/gin.(*Context).Next\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185\ngithub.com/gin-gonic/gin.(*Engine).handleHTTPRequest\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/gin.go:633\ngithub.com/gin-gonic/gin.(*Engine).ServeHTTP\n\t/Users/ninesun/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/gin.go:589\nnet/http.serverHandler.ServeHTTP\n\t/Users/ninesun/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.4.darwin-arm64/src/net/http/server.go:3210\nnet/http.(*conn).serve\n\t/Users/ninesun/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.4.darwin-arm64/src/net/http/server.go:2092"}
2025/02/24 10:42:28 encoding for model: no encoding for model qwen-plus
2025/02/24 10:42:28 encoding for model: no encoding for model qwen-plus
Observation: Tool kubectl failed with error {
    "apiVersion": "v1",
    "items": [],
    "kind": "List",
    "metadata": {
        "resourceVersion": ""
    }
}
Error from server (NotFound): pods "|" not found
Error from server (NotFound): pods "jq" not found
Error from server (NotFound): pods "'.items[]" not found
Error from server (NotFound): pods "|" not found
Error from server (NotFound): pods "{name:" not found
Error from server (NotFound): pods ".metadata.name," not found
Error from server (NotFound): pods "labels:" not found
Error from server (NotFound): pods ".metadata.labels," not found
Error from server (NotFound): pods "image:" not found
Error from server (NotFound): pods ".spec.containers[].image," not found
Error from server (NotFound): pods "startTime:" not found
Error from server (NotFound): pods ".status.startTime}'" not found. Considering refine the inputs for the tool.

```
cmd := exec.Command("kubectl", strings.Split(command, " ")...)
这个函数使用 Go 的 exec.Command 执行 kubectl 命令，假设命令以空格分隔为参数。
它不支持管道（|）或 shell 特定的语法（如 grep），因为 exec.Command 是直接调用 kubectl 的子进程，而非 shell 环境。


11. 已经有结果了，因为解析失败，导致resp又重新需要喂给LLM做总结
```json
Unable to parse tools from LLM (invalid character '\n' in string literal), summarizing the final answer.
```
2025年03月12日19:29:51 如何优化LLM 输出的结果呢？如何避免这次chat请求，来提升性能并节省token

12. 对于用户模糊的提问，LLM如何引导用户？ (暂不支持)
    例如用户提问：iotdb 版本是什么?  这个对大模型来说会增加很多的token消耗，如何引导用户提供更多的信息，来减少token消耗呢？
    需要保存上下文，来引导用户提供更多的信息，这个需要在chat的时候保存上下文，然后在下次chat的时候引导用户提供更多的信息。

## prompt 优化 (已完成)
### 避免全量输出-o json 或者-o yaml
大模型好像没有遵循我的prompt，总是会kubectl get nodes -o json，或kubectl get po -o json。 
这个操作会产生大量的数据，超过上下文窗口。目前定义的max_token是2048
kubectl get pods/node/deploy/statefulset  -o json


## releaseNote
