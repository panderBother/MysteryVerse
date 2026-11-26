# Task 11 实施与验证报告

> 日期：2026-11-25
> 状态：DONE
> DQ-18：APPROVED / IMPLEMENTED / VERIFIED
> DQ-19：APPROVED / IMPLEMENTED / TEST-VERIFIED；生产基础设施 NOT VERIFIED

## 1. 批准与范围

负责人此前要求先明确批准 DQ-18/DQ-19，随后于 2026-11-25 回复“开始执行吧”。
该指令被记录为对两项候选推荐方案的批准，不扩展到 DQ-20 至 DQ-22 或
ADR-001/004/005/008。

本任务没有：

- 开放外部 MCP。
- 实现正式 Run/Checkpoint 持久化。
- 接入真实模型 Provider。
- 修改权威游戏状态或 AI DM 业务。
- 冻结阶段 DSL、投票策略、SecretPayload AAD、ViewerManifest 或 Memory 协议。

## 2. DQ-18 实施结果

### 2.1 Registry

- `packages/contracts/json-schema/mysteryverse.schema.json` 仍是字段定义唯一源。
- `packages/contracts/registry/schema-governance.json` 保存版本、Active、
  compatibility、lifecycle、Replay 和 runtime binding 治理策略。
- 生成器产出 `schema-registry.json`，覆盖当前 142 个定义。
- 每个定义包含 Schema ID、name、version、type、status、decision status、
  compatibility、owner、时间、definition hash 和 version history。

### 2.2 Version、Active 与兼容性

- Registry 使用 SemVer；初始版本为 `1.0.0`。
- 被历史 Event/Run 引用的版本不能被删除。
- `x-contract-status: accepted` 才能成为 Active；其余保持 experimental。
- 同一 Schema 最多一个 Active Version。
- 自动兼容检查覆盖字段/required/type/$ref/enum/default/nested/composed changes。
- 破坏性变化会使生成检查失败；兼容新增产生 warning。
- 基线固定为 `compatibility/schema-baseline-v1.json`，不能由日常 CI 自动覆盖。

### 2.3 Runtime 输出绑定

- 新增正式 `DemoAgentOutput` 契约和 fixture。
- 当前唯一 Active 输出绑定：`demo_result -> DemoAgentOutput@1`。
- Python `ContractsSchemaRegistry` 验证 Active 状态和绑定。
- `OutputSchemaRegistry` 从生成模型注册 Pydantic 严格校验器；模型不能选择 Schema。
- 其他业务 Agent Schema 仍为 experimental/blocked。

## 3. DQ-19 实施结果

### 3.1 mTLS

- FastAPI/Uvicorn 支持显式 server certificate、private key、client CA 和
  required client certificate。
- Go Internal AI Client 支持 Root CA、client certificate/key、ServerName 和
  TLS 1.2 最低版本。
- Go 测试覆盖双向证书成功、缺失客户端证书和过期客户端证书。
- 真实 CA、吊销、部署和生产端到端仍为 `NOT VERIFIED`。

### 3.2 HMAC 请求完整性

- 正式 profile：`mv-hmac-sha256-v1`。
- Canonical 覆盖 signature version、Key ID、timestamp、nonce、Service ID、
  method、path、query 和 body SHA-256。
- 使用 Key ID 对应密钥和恒定时间比较。
- 支持时间窗口、body/path/query 篡改拒绝和错误 envelope。

### 3.3 Key ID 与 KMS 抽象

- Key ID 有格式校验，不属于秘密。
- 支持 Active/Previous 轮换与 revoked key 拒绝。
- Python 定义 Key Provider/KMS 抽象和环境隔离实现。
- 生产配置要求 external KMS；当前 external provider 未实现，因此生产状态为
  `NOT VERIFIED`。

### 3.4 Shared Nonce

- `NonceStore` 是可替换接口。
- 内存 Store 用于开发和单进程测试。
- Redis Store 使用原子 `SET NX EX`，支持 TTL。
- 跨 App 重放测试和并发原子消费测试通过。
- Nonce Store 故障返回 `NONCE_STORE_UNAVAILABLE`，不会放行请求。

### 3.5 ACL Scope

- 认证身份为 Service ID + Token + 可选签名身份。
- 授权 scope 分为 operation、resource 和 tenant。
- 当前 Agent endpoint 要求 `ai:agent:run` 与 `run:<run_id>`。
- 有租户上下文时还必须包含 `tenant:<tenant_id>`。
- 无效 scope、资源不匹配和跨租户访问均 deny by default。

### 3.6 安全审计

- 记录 Service ID、Request ID、Key ID、action、authentication result、
  authorization result、scopes 和 failure category。
- 不记录 token、签名密钥、私密正文或完整 payload。
- 日志字段和脱敏测试通过。

## 4. 验证结果

最终执行：

```powershell
./scripts/mv.ps1 check
```

结果：

- Monorepo structure check：通过。
- Contracts generation/validation：142 定义，通过。
- Contract registry tests：11 项通过。
- TypeScript contract tests：27 项通过。
- Database/sqlc generation check：通过。
- Web ESLint、TypeScript、Prettier：通过。
- Go vet/test/build：通过。
- Go mTLS Test：通过，包含 missing/expired client certificate。
- Python pytest：48 项通过。
- Python Ruff：通过。
- Python mypy：53 个源文件通过。
- Web production build：通过。
- Python sdist/wheel build：通过。

## 5. 状态回答

- DQ-18 是否 IMPLEMENTED：是。
- DQ-18 是否 VERIFIED：是，测试与生成漂移验证通过。
- DQ-19 是否 IMPLEMENTED：是。
- DQ-19 是否 VERIFIED：测试范围 VERIFIED；生产基础设施 NOT VERIFIED。
- 是否可进入 DQ-21：Task 11 完成时不可以；负责人随后已单独批准 DQ-21，
  当前实施状态见 `task-12-report.md`。
- 是否仍禁止 AI DM 正式实现：是。具体业务 Schema、ADR-008、DQ-21、DQ-22
  和 GATE-AI 仍未满足。

## 6. 剩余风险

- 真实 CA/KMS/Redis 多实例未部署。
- 证书身份与 `X-Service-Id` 的生产级映射仍依赖部署网关/证书策略验证。
- 生产证书吊销、自动轮换、时钟偏差监控和容量测试未执行。
- 外部 MCP、Run/Checkpoint、真实 Provider 和正式业务 Agent Schema 仍被门禁阻塞。
