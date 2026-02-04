# Orchids-2api 文档

## 项目简介

**Orchids-2api** (orchids-api) 是一个 Go 语言编写的 API 代理服务器，提供多账号管理与负载均衡代理功能，兼容 Claude API 格式的请求转发。

### 核心功能

- 多账号管理与负载均衡代理
- 兼容 Claude API 格式的请求转发
- 将请求代理到 Orchids 后端服务
- 提供 Web 管理界面
- 支持多种 AI 模型

### 支持的模型

| 模型 | 提供商 | 描述 |
|------|--------|------|
| Gemini 3 Flash | Google | 快速高效的 Gemini 模型 |
| Claude Opus 4.5 | Anthropic | Anthropic 最强大的 Claude 模型 |
| Claude Sonnet 4.5 | Anthropic | 均衡的 Claude 模型 |
| GPT-5.2 Codex | OpenAI | OpenAI 高级代码生成模型 |

## 文档目录

| 文档 | 描述 |
|------|------|
| [架构设计](./docs/architecture.md) | 目录结构、核心组件、请求流程、数据模型 |
| [API 接口](./docs/api-reference.md) | 所有端点列表、请求/响应格式、认证说明 |
| [部署指南](./docs/deployment.md) | Docker 构建、本地开发、生产部署 |
| [配置说明](./docs/configuration.md) | 环境变量、配置文件格式 |

## 快速开始

```bash
# 本地开发
go mod download
go run ./cmd/server/main.go

# Docker 部署
./build.sh
docker compose up -d
```

## 主要特性

1. **多账号管理** - 支持添加、编辑、删除多个 Orchids 账号
2. **多策略负载均衡** - 支持加权随机、轮询、最少连接三种策略
3. **故障转移** - 账号失败时自动切换
4. **健康检查** - 自动追踪账号健康状态，排除不健康账号
5. **速率限制** - 每账号请求速率限制保护
6. **多模型支持** - 支持 Gemini 3 Flash、Claude Opus 4.5、Claude Sonnet 4.5、GPT-5.2 Codex
7. **模型别名** - 透明映射常用模型名称到实际模型
8. **模型列表 API** - OpenAI 兼容的 /v1/models 端点
9. **工具调用** - 完整支持 Claude Tool Use
10. **流式响应** - SSE 实时响应
11. **Token 计数** - 估算输入/输出 Token
12. **调试日志** - 详细的请求/响应日志
13. **管理界面** - Web UI 管理账号
14. **导入导出** - 账号配置备份恢复

## API 端点

### 主要端点

- `GET /v1/models` - 获取可用模型列表
- `POST /v1/messages` - Claude API 代理端点
- `GET /health` - 健康检查

### 负载均衡管理

- `GET /api/loadbalancer/stats` - 获取负载均衡器统计
- `GET /api/loadbalancer/health` - 获取账号健康状态
- `PUT /api/loadbalancer/strategy` - 设置负载均衡策略

详细 API 文档请参阅 [API 接口](./docs/api-reference.md)。
