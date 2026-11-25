# Contracts Schema Registry

> 状态：APPROVED / IMPLEMENTED / VERIFIED
> 决策：DQ-18
> 日期：2026-11-25
> 审批证据：负责人在要求先批准 DQ-18/DQ-19 后，于 2026-11-25 明确下达
> “开始执行吧”。本次批准仅覆盖本文件的候选治理方案和 DQ-19 安全基础，不包含
> DQ-20 至 DQ-22。

## 1. 唯一权威源

正式 Schema 定义仍位于：

```text
packages/contracts/json-schema/mysteryverse.schema.json
```

治理元数据位于：

```text
packages/contracts/registry/schema-governance.json
```

治理元数据只描述版本、状态、生命周期和运行时绑定，不复制字段定义。字段定义的
唯一权威源仍是 `mysteryverse.schema.json`。

## 2. 生成物

`scripts/contracts.ps1 generate` 生成并校验：

- Go、TypeScript、Python 三种语言类型。
- `packages/contracts/registry/schema-registry.json`。
- AI Runtime 使用的 Registry 副本。
- `active_output_schemas.py` 运行时绑定。
- Schema 指纹和首个兼容性基线。

生成物禁止手工修改。`scripts/contracts.ps1 check` 会重新生成并检测漂移。

## 3. Registry 模型

每个定义都包含：

- `schema_id`：稳定 Registry ID。
- `name` / `definition`：JSON Schema 定义名和引用。
- `version` / `active_version`：Registry SemVer。
- `schema_type`：`public|internal|runtime|persistence|experimental`。
- `status`：`draft|experimental|active|deprecated|retired`。
- `decision_status`：`accepted|proposed|pending`。
- `compatibility`：当前兼容策略。
- `owner`：维护服务。
- `created_at` / `deprecated_at` / `sunset_at`。
- `definition_sha256`：定义内容指纹。
- `version_history`：追加式版本历史。

Schema 只有在源定义标记 `x-contract-status: accepted` 时才进入 `active`。其他定义
进入 `experimental`，不得被 Runtime 当作可信业务输出。

## 4. 版本与 Active 规则

- Registry 版本使用 SemVer。
- 不兼容变更新增 Major。
- 兼容的字段新增新增 Minor。
- 仅治理元数据变化使用 Patch。
- 同一 Schema 最多一个 Active Version。
- 客户端和模型不能自行决定默认 Active Version。
- Runtime 必须显式读取已审批 Schema 绑定。
- Active 回滚只切换默认指针，不删除旧版本和历史证据。
- 当前不允许租户级或环境级 Schema 覆盖。

## 5. 兼容性门禁

自动检查覆盖：

- 字段新增、删除与必填/可选变化。
- 类型与 `$ref` 变化。
- Enum 新增和删除。
- 默认值变化。
- 数组 Item 与组合 Schema 分支变化。
- 嵌套对象递归变化。

删除字段、提升必填、删除 Enum、改变类型或 `$ref` 属于破坏性变更，会使
`contracts.ps1 check` 失败。可选的兼容新增会产生 warning，便于评审，但不直接
阻断。

首个基线保存在：

```text
packages/contracts/compatibility/schema-baseline-v1.json
```

新基线必须经过正式契约评审，不能为了通过 CI 自动覆盖。

## 6. Replay 与生命周期

- 历史事件和 Run 必须保存实际 Schema key/version，不能只依赖 Git commit。
- 缺版本、未知版本和无法解析版本必须 fail-closed。
- 历史迁移需要显式 Reader Migration，不允许静默丢弃字段。
- 被历史事件或 Run 引用的 Schema 不得删除。
- Retired Schema 仍可只读用于历史 Replay 和审计。
- 游戏事件、Memory 和 WebSocket 的冻结分别继续受 ADR-004、ADR-001、ADR-002
  阻塞；本 Registry 不解除这些门禁。

## 7. Runtime 绑定

当前唯一正式 Active 输出绑定为：

```text
demo_result -> DemoAgentOutput@1
```

Python `ContractsSchemaRegistry` 在运行时验证绑定指向 Active Schema。
`OutputSchemaRegistry` 再从生成模型注册严格 Pydantic 校验器。模型不能选择
Schema key/version 或 Active 状态。

AI DM、NPC、规则解释和剧本生成仍需各自业务 Schema 与 Go 二次规则校验；本任务
没有提前冻结这些 Schema。

## 8. 验证命令

```powershell
./scripts/contracts.ps1 generate
./scripts/contracts.ps1 check
uv run --project ai_service pytest packages/contracts/tests
```
