# libcortex 对接文档

本文档说明 `cmd/libcortex` 暴露给上层（C/Swift/Java 等）的接口、参数与事件，方便在 App 侧完成集成。

## 导出函数

- `InitSDK(configPath *C.char) -> *C.char`
  - 作用：初始化配置管理器与运行时；会重建默认运行时。
  - `configPath`：如果是目录则读取/创建该目录下的 `config.json`；如果是以 `.json` 结尾的文件路径则直接使用；为空时落到 `${UserConfigDir}/CortexGo/config.json`。
  - 返回：`"Success"` 或 `"Error: <message>"` 字符串，需要由调用方使用 `FreeString` 释放。
- `RegisterCallback(cb C.EventCallback)`
  - 作用：注册全局事件回调，签名为 `void (*cb)(char* topic, char* payload)`。
  - 回调时 `topic`/`payload` 由 Go 创建，生命周期归 Go 管理；只需对 `InitSDK`/`Call` 等返回值调用 `FreeString`。
- `UpdateConfig(jsonStr *C.char) -> *C.char`
  - 作用：以 JSON（`Config` 结构）覆写配置文件并应用。
  - 返回同 `InitSDK`。
- `GetConfig() -> *C.char`
  - 作用：获取当前配置的 JSON 文本，字段见下文。
- `Call(method *C.char, params *C.char) -> *C.char`
  - 作用：统一 RPC 入口；`method` 为字符串，`params` 为 JSON 字符串。
  - 返回值结构：`{"code":int,"msg":string,"data":any}`，成功 `code=200`。
- `FreeString(str *C.char)`
  - 作用：释放由 Go 分配并返回给 C 侧的字符串。

## Config 字段（`config/config.go`）

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `project_dir` | string | 工作目录 | 项目根路径 |
| `results_dir` | string | `<project_dir>/results` | 生成报告/历史 Markdown 的目录 |
| `data_dir` | string | `<project_dir>/data` | 数据存放目录 |
| `data_cache_dir` | string | `<project_dir>/data/cache` | 数据缓存目录 |
| `eino_debug_enabled` | bool | `false` | 是否开启 Eino 调试 |
| `eino_debug_port` | int | `52538` | 调试端口 |
| `cache_enabled` | bool | `true` | 是否启用缓存 |
| `longport_app_key` / `longport_app_secret` / `longport_access_token` | string | 空 | Longport API 认证信息 |
| `deepseek_api_key` | string | 空 | DeepSeek Chat API Key，`agent.stream` 必填 |

> 支持通过环境变量覆盖：`CACHE_ENABLED`、`EINO_DEBUG_ENABLED`、`EINO_DEBUG_PORT`、`LONGPORT_*`、`DEEPSEEK_API_KEY`。

## Call 方法列表

统一返回 `{"code":int,"msg":string,"data":...}`，失败时 `code` 为 `500/404` 等，`msg` 含错误原因。

- `system.info`
  - 入参：无（`params` 可为空字符串）。
  - 出参 `data`：`{"version":"1.0.0","os":"android/ios"}`。

- `agent.stream`
  - 入参 JSON（`models.AgentInitParams`）：
    - `symbol` (string, 必填)：交易标的。
    - `trade_date` (string, 可选)：`YYYY-MM-DD`，默认当天。
    - `prompt` (string, 可选)：自定义提示词，默认 `Analyze trading opportunities for <symbol> on <trade_date>`。
  - 前置要求：`deepseek_api_key` 必填；`trade_date` 可解析；`symbol` 非空。
  - 出参 `data`：`{"status":"started"}`。实际编排在后台 goroutine 运行，后续进度通过回调事件推送（见下节）。
  - 结束事件：成功时触发 `agent.finished`，异常时 `agent.error`。

- `agent.history.list`
  - 入参 JSON（`models.HistoryParams`），可为空：
    - `cursor` (string, 可选)：上一页返回的 `session_id` 书签（为空表示第一页）。
    - `limit` (int, 可选)：每页数量，默认 50，最大 200。
  - 前置要求：`data_dir` 已配置；使用 `data_dir/agent.db` 中的会话记录。
  - 出参 `data`（`models.HistoryListResponse`）：
    - `items`: `[{session_id,symbol,trade_date,prompt,status,created_at,updated_at}]`
    - `next_cursor`: string，下一页游标（同 `session_id` 书签）；无则为空。
    - `has_more`: bool。

- `agent.history.info`
  - 入参 JSON（`models.HistoryInfoParams`）：
    - `session_id` (string, 必填)：`agent.history.list` 返回的 `session_id`。
  - 出参 `data`（`models.HistoryInfoResponse`）：
    - `session`: `{session_id,symbol,trade_date,prompt,status,created_at,updated_at}`
    - `messages`: `[{id,role,agent,content,status,finish_reason,seq,created_at,updated_at}]`（按 seq 升序）。

## 事件回调（`RegisterCallback`）

`agent.stream` 会通过 `bridge.Notify` 触发事件，`topic` 统一以 `agent.` 前缀；`payload` 为 JSON 序列化的 `models.ChatResp` 或错误信息：

- `agent.message_chunk`：AI 回复的分片事件；`payload.content` 为最新文本片段，`payload.tool_calls` 可能包含工具调用参数片段。
- `agent.tool_call_result_final`：工具执行完成后的消息（最终态），包含 `tool_call_id`、`tool_name` 及结果文本。
- `agent.text_final`：一次完整的助手回复聚合结果（文本与工具调用合并），落盘时使用该事件。
- `agent.error`：流执行出错；若来自模型回调则 `payload` 是 `models.ChatResp`（`role=system`），若是整体流程失败则 `payload` 形如 `{"error": "<message>"}`。
- `agent.finished`：流程正常结束，`payload={"status":"completed"}`。

回调内容均为 UTF-8 JSON 文本，上层可按需解析并展示。
