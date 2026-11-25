# 服务边界

> 状态：Candidate Baseline，依赖 Proposed ADR
> 依据：ADR-001、ADR-004、ADR-008、ADR-009

## 1. 单写原则

同一业务事实只有一个权威写入者。写权限按领域划分，不按调用方便划分。

| 领域 | 权威写入者 | Python 访问方式 |
| --- | --- | --- |
| 用户、Session、资料 | Go | 不访问 |
| 剧本、版本、发布 | Go | Artifact/校验候选 |
| 房间、席位、邀请 | Go | 不访问 |
| GameState、GameEvent、投票、结算 | Go | 授权读取或提交候选 |
| NPC 关系、Memory 元数据 | Go | Candidate + 授权切片 |
| Prompt、AI Run、Usage、Checkpoint | Python `mv_ai` | 自有 schema |
| Admin、举报、处置、审计 | Go | 不访问 |

## 2. Go 边界

Go 负责：

- 认证、授权和 Session。
- 房间、内容、工坊和社区业务。
- 游戏规则、状态机、Timer 和结算。
- 幂等与版本控制。
- 事件持久化、快照、重连和投影。
- ContextProjector 和 AI Gateway。
- MemoryCandidate 最终校验与写入。
- Artifact 注册和发布原子切换。
- 运营后台和审计。

Go Handler 只负责协议转换、认证上下文、Schema 校验和错误映射。业务必须进入
Application、Domain、Engine 或明确的后台服务。

## 3. Python 边界

Python 负责：

- 文档解析和结构化。
- 角色、场景、线索、任务和剧情生成。
- 确定性辅助校验、语义评审和有限修复。
- AI DM、NPC 对话、AI 玩家候选。
- MemoryCandidate、复盘文本和引用。
- Prompt、Provider、模型路由、Usage 和 trace。

Python 不得：

- 直接写或锁定 `mv_core` 权威表。
- 直接发布内容或广播正式事件。
- 通过额外查询扩大 Go 授权切片。
- 使用客户端传入的实体 ID 猜测未授权对象。
- 生成可执行规则、SQL、代码或动态路由。

## 4. 上下文职责

| 步骤 | 负责方 |
| --- | --- |
| 身份、房间、席位、角色、阶段校验 | Go |
| 对象与字段 ACL | Go |
| reveal gate 与事件可见性 | Go |
| AllowedKnowledgeSlice | Go |
| 在切片内排序、摘要和 token 裁剪 | Python |
| Context Manifest 授权部分 | Go |
| Context Manifest 注入顺序与预算 | Python |
| 输出 Schema、规则冲突和泄漏复检 | Go |

同步 AI 回合由 Go 在调用前完成上下文组装。Python 不得在同一同步回合内再次
回调 Go Context Resolve，避免嵌套调用环。

## 5. 内部调用

允许方向：

- Go → Python：DM/NPC/意图解析/规则解释。
- Go → Redis Streams → Python Worker：长任务。
- Python → Go：Artifact 注册、进度、MemoryCandidate、意图校验/提交。
- Python → Go：受限事实或已授权上下文读取。

所有调用必须带：

- 服务身份。
- `trace_id`。
- `run_id` 或 `job_id`。
- `schema_version`。
- `Idempotency-Key`。
- deadline。

## 6. Memory 边界

Python 生成 `MemoryCandidate`。Go 校验并写 `npc_memories`、关系变化和加密摘要。
Python 读取的是 Go 裁剪后的 Memory View。

`mv_ai.memory_candidates` 可以作为非权威候选/运行记录，但不得被游戏规则查询为事实。

## 7. PlayerContext 边界

MVP 不持久化 PlayerContext。每次 Agent 执行从服务端可见投影重建。
AI Run trace 可以记录 context manifest、来源 hash 和预算，但不得保存完整 D2-D4
上下文明文。

## 8. 异步增强

图片生成与 AI 复盘点评通过长任务执行：

- 不占用实时 AI 并发池。
- 不参与规则状态事务。
- 失败不阻断文本候选、游戏或结果。
- 可关闭且受配额控制。

## 9. 反模式

- Handler 内直接拼业务规则。
- Python 直接 SQL 查询 Go 表。
- Redis 被当作唯一状态。
- 客户端自报权限或可见字段。
- AI 输出直接写事件或结果。
- 为未来模式提前拆多个网络微服务。

