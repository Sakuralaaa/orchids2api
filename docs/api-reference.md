# API 接口文档

## 端点概览

| 端点 | 方法 | 描述 | 认证 |
|------|------|------|------|
| `/v1/messages` | POST | Claude API 代理端点 | 无 |
| `/v1/models` | GET | 获取可用模型列表 | 无 |
| `/v1/models/{id}` | GET | 获取单个模型信息 | 无 |
| `/api/accounts` | GET | 获取所有账号列表 | Basic Auth |
| `/api/accounts` | POST | 创建新账号 | Basic Auth |
| `/api/accounts/{id}` | GET | 获取单个账号 | Basic Auth |
| `/api/accounts/{id}` | PUT | 更新账号 | Basic Auth |
| `/api/accounts/{id}` | DELETE | 删除账号 | Basic Auth |
| `/api/export` | GET | 导出账号数据 (JSON) | Basic Auth |
| `/api/import` | POST | 导入账号数据 (JSON) | Basic Auth |
| `/api/loadbalancer/stats` | GET | 获取负载均衡器统计信息 | Basic Auth |
| `/api/loadbalancer/health` | GET | 获取账号健康状态 | Basic Auth |
| `/api/loadbalancer/strategy` | GET/PUT | 获取/设置负载均衡策略 | Basic Auth |
| `/health` | GET | 健康检查 | 无 |
| `{ADMIN_PATH}/*` | GET | 管理界面 | Basic Auth |

## 认证

### 管理接口认证

- **类型**: HTTP Basic Authentication
- **保护端点**: `/api/*`, 管理界面
- **凭据**: `ADMIN_USER` / `ADMIN_PASS`

### 上游 API 认证

- **类型**: Bearer Token (JWT)
- **流程**:
  1. 使用账号的 ClientCookie 调用 Clerk API
  2. 获取 SessionID
  3. 生成 JWT Token
  4. 作为 Authorization Header 发送到上游

## /v1/models 端点

### 获取可用模型列表

```bash
GET /v1/models
```

响应格式 (OpenAI 兼容):

```json
{
  "object": "list",
  "data": [
    {
      "id": "gemini-3-flash",
      "object": "model",
      "owned_by": "google"
    },
    {
      "id": "claude-opus-4.5",
      "object": "model",
      "owned_by": "anthropic"
    },
    {
      "id": "claude-sonnet-4.5",
      "object": "model",
      "owned_by": "anthropic"
    },
    {
      "id": "gpt-5.2-codex",
      "object": "model",
      "owned_by": "openai"
    }
  ]
}
```

## /v1/messages 端点

### 请求格式

兼容 Claude API 格式：

```json
{
  "model": "claude-opus-4.5",
  "messages": [
    {
      "role": "user",
      "content": "Hello"
    }
  ],
  "stream": true,
  "tools": []
}
```

### 响应格式

SSE 流式响应，兼容 Claude API 格式。

### 支持的模型

| 模型 ID | 名称 | 提供商 | 描述 |
|---------|------|--------|------|
| `gemini-3-flash` | Gemini 3 Flash | Google | 快速高效的 Gemini 模型 |
| `claude-opus-4.5` | Claude Opus 4.5 | Anthropic | Anthropic 最强大的 Claude 模型 |
| `claude-sonnet-4.5` | Claude Sonnet 4.5 | Anthropic | 均衡的 Claude 模型 |
| `gpt-5.2-codex` | GPT-5.2 Codex | OpenAI | OpenAI 高级代码生成模型 |

### 模型别名映射

| 请求模型 | 映射到 |
|----------|--------|
| `opus`, `claude-opus`, `claude-3-opus-*` | `claude-opus-4.5` |
| `sonnet`, `claude-sonnet`, `claude-3-sonnet-*` | `claude-sonnet-4.5` |
| `haiku`, `claude-3-haiku-*` | `gemini-3-flash` |
| `gpt-4`, `gpt-4-turbo`, `gpt-4o`, `gpt-5`, `codex` | `gpt-5.2-codex` |
| `gemini`, `gemini-pro`, `gemini-flash` | `gemini-3-flash` |

## 负载均衡器 API

### 获取负载均衡器统计

```bash
GET /api/loadbalancer/stats
Authorization: Basic <credentials>
```

响应:

```json
{
  "strategy": "weighted_random",
  "healthy_count": 5,
  "unhealthy_count": 1,
  "health_enabled": true
}
```

### 获取账号健康状态

```bash
GET /api/loadbalancer/health
Authorization: Basic <credentials>
```

响应:

```json
{
  "1": {
    "ID": 1,
    "FailureCount": 0,
    "LastSuccess": "2024-01-15T10:00:00Z",
    "IsHealthy": true,
    "ConsecutiveFails": 0,
    "TotalRequests": 150
  }
}
```

### 设置负载均衡策略

```bash
PUT /api/loadbalancer/strategy
Authorization: Basic <credentials>
Content-Type: application/json

{
  "strategy": "round_robin"
}
```

支持的策略:
- `weighted_random` - 加权随机 (默认)
- `round_robin` - 轮询
- `least_connections` - 最少连接
