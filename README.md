# CortexGo

CortexGo 是一个基于 CloudWeGo Eino 的多智能体交易分析引擎(TradeAgentEngine)，同时提供 c-shared 动态库（libcortex）供 App 侧集成。系统围绕交易标的串联分析、辩论、交易与风控流程，支持流式回调、结果落盘与历史追踪。

## 功能概览
- 多阶段编排：市场/社交/新闻/基本面分析 → 多空辩论 → 交易 → 风险评审
- 可插拔工具：Longport 行情、技术指标、Google News、Reddit
- 流式事件回调 + SQLite 历史记录
- Markdown 报告落盘（`results/<symbol>/<trade_date>/`）
- 配置热更新与本地缓存（`data/cache`）

## 编排流程
```
Market Analyst -> Social Analyst -> News Analyst -> Fundamentals Analyst
Bull Researcher <-> Bear Researcher (max 2 rounds) -> Research Manager
Trader -> Risky Analyst -> Safe Analyst -> Neutral Analyst (max 3 rounds) -> Risk Judge
```

## 快速开始

### 运行 Demo
1. 准备环境变量
   - `DEEPSEEK_API_KEY` (必填)
   - `LONGPORT_APP_KEY` / `LONGPORT_APP_SECRET` / `LONGPORT_ACCESS_TOKEN` (可选，缺省使用 mock 行情)
2. 运行
   - `go run cmd/demo/main.go`
3. 结果
   - Markdown 报告：`results/<symbol>/<trade_date>/`
   - 历史记录：`data/agent.db`

### 构建 libcortex 动态库
- macOS Universal:
  - `./scripts/build_libcortexgo.sh`
  - 输出：`build/libcortex.dylib`、`build/libcortex.h`
- 其他平台参考：
  - `go build -buildmode=c-shared -o build/libcortex.so ./cmd/libcortex/...`

### C/Swift/Java 接口
导出函数：`InitSDK`、`RegisterCallback`、`UpdateConfig`、`GetConfig`、`Call`、`FreeString`。  
RPC 方法：`system.info`、`agent.stream`、`agent.history.list`、`agent.history.info`。  
完整参数与事件说明见 `doc.md`。

## 配置
如果是Lib库集成，使用Json配置文件，进行初始化。
默认配置路径：`${UserConfigDir}/CortexGo/config.json`（`InitSDK` 可传入自定义目录或文件）。  

如果是测试Demo，配置env文件，`cp .env.example .env`，在`.env`文件里面配置DeepSeek的APIKey，长桥证券的OpenAPI Key等信息。
支持环境变量覆盖：`CACHE_ENABLED`、`EINO_DEBUG_ENABLED`、`EINO_DEBUG_PORT`、`LONGPORT_*`、`DEEPSEEK_API_KEY`。

常用字段：
- `project_dir` / `results_dir` / `data_dir` / `data_cache_dir`
- `eino_debug_enabled` / `eino_debug_port` / `cache_enabled`
- `longport_app_key` / `longport_app_secret` / `longport_access_token`
- `deepseek_api_key`

## 目录结构
```
cmd/
  demo/        # 本地演示入口
  libcortex/   # c-shared 动态库入口
internal/
  agents/      # 各类 agent 实现
  graph/       # 编排图与回调
  tools/       # 市场/新闻/社交工具
  storage/     # SQLite 持久化
config/        # 配置管理与热更新
pkg/
  dataflows/   # 数据源与缓存
  app/         # runtime/engine
  bridge/      # 回调桥接
```

## 依赖
- Go 1.24+
- CloudWeGo Eino
- DeepSeek（OpenAI 兼容接口）
- LongPortOpenAPI (长桥证券的OpenAPI)
