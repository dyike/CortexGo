# CortexGo Eino Agent Orchestration

基于CloudWeGo Eino框架实现的交易agent编排系统，参考了deer-go项目的设计模式。

## 项目结构

```
pkg/eino/
├── infrastructure.go  # 模型基础设施和Mock实现
├── state.go          # 交易状态管理
├── builder.go        # 主编排器构建
├── coordinator.go    # 协调器agent
├── analyst.go        # 分析师agent
├── researcher.go     # 研究员agent  
├── trader.go         # 交易员agent
├── risk_manager.go   # 风险管理agent
└── reporter.go       # 报告生成agent

consts/
└── nodes.go          # agent节点常量定义

cmd/
└── main.go      # 主程序入口
```

## 架构设计

### 1. Agent编排流程

```
分析师阶段: Market → Social → News → Fundamentals
研究阶段: Bull Researcher ↔ Bear Researcher → Research Manager
交易阶段: Trader → 风险分析 (Risky ↔ Safe ↔ Neutral) → Risk Judge
```

```
START → Coordinator → Analyst → Researcher → Trader → RiskManager → Reporter → END
```

每个agent完成任务后，通过状态管理决定下一个要执行的agent。

### 2. 核心组件

#### TradingState
- 维护整个交易流程的状态
- 包含消息历史、市场数据、分析报告、交易决策等
- 通过Goto字段控制agent流转

#### Agent节点
每个agent都是一个Eino Graph，包含：
- `load`: 加载消息和上下文
- `agent`: 执行核心逻辑（使用ChatModel）
- `router`: 路由到下一个agent

#### 编排器(Builder)
- 创建所有agent子图
- 配置agent之间的连接关系
- 管理整体执行流程

### 3. Agent功能

- **Coordinator**: 分析用户需求，决定激活哪个agent
- **Analyst**: 进行技术分析，生成交易建议
- **Researcher**: 进行基本面研究和市场调研
- **Trader**: 基于分析做出具体交易决策
- **RiskManager**: 评估风险，批准或拒绝交易
- **Reporter**: 生成最终的交易报告

## 使用方法

### 1. 控制台模式

```bash
go run cmd/eino_main.go
```

程序会提示输入交易符号和需求，然后自动执行agent编排流程。

### 2. 服务器模式

```bash
go run cmd/eino_main.go -s
```

（服务器模式待实现）

## 技术特点

### 1. 基于Eino框架
- 使用compose.Graph构建agent流程图
- 支持状态管理和上下文传递
- 内置消息路由和流程控制

### 2. 模块化设计
- 每个agent独立实现，易于扩展
- 统一的状态管理和消息传递
- 清晰的agent职责划分

### 3. Mock实现
- 当前使用Mock ChatModel进行演示
- 可以轻松替换为真实的LLM模型
- 支持工具调用模拟

## 扩展说明

### 添加新Agent
1. 在`consts/nodes.go`中定义新的节点常量
2. 创建新的agent文件，实现对应的Graph
3. 在`builder.go`中注册新的agent节点
4. 更新其他agent的路由逻辑

### 集成真实LLM
1. 替换`infrastructure.go`中的MockChatModel
2. 使用Eino提供的真实模型组件
3. 配置API密钥和模型参数

### 添加工具支持
1. 定义工具的schema.ToolInfo
2. 在对应agent中绑定工具
3. 实现工具调用的处理逻辑

## 依赖

- `github.com/cloudwego/eino v0.3.40`
- Go 1.23.6+

## 参考

本实现参考了`/Users/ityike/Code/go/src/github.com/dyike/eino-examples/flow/agent/deer-go`项目的设计模式，采用了相同的编排思路和状态管理方式。
