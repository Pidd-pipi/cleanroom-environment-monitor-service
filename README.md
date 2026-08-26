# cleanroom-environment-monitor-service

基于 Go 实现的**半导体洁净室环境监控全栈 Web 项目**（Go 后端服务 + `go:embed` 内嵌原生前端），完成洁净区分区管理、粒子/温湿度采集、ISO 洁净度等级判定、设备联锁与报警处置。项目为干净基线，无故意埋错，适合作为跨文件协同改动的评测基准，也具备企业级可部署形态（结构化日志、安全中间件、原子持久化、优雅关闭、Docker 镜像与健康检查）。

## 一、功能与业务规则

- **洁净区状态机**：`normal → elevated → over_limit → interlocked → restored`（另支持 `restored → interlocked` 反向联动，以满足整区一致规则）；超限（粒子比 ≥ 1.5×）立即下发联锁，持续偏高/超限 10 分钟未恢复由定时任务自动联锁通风。
- **ISO 洁净度判定**：按 ISO 14644-1 双粒径（≥0.3μm、≥0.5μm）对照表判定等级，任一粒径超标取较差等级；超过表范围标记 `over_table`。
- **数据有效性**：粒子计数器处于 PM 维护或标定过期时，上报数据标记 `invalid` 且不参与等级判定；近 50 条样本无效占比 > 30% 产生 `data_quality`「数据可信度不足」报警。
- **联锁（整区一致）**：粒子浓度 ≥ 上限 1.5 倍自动下发 FFU 提速/新风加大/排风增强指令并记录联锁日志；同一物理区域的所有洁净区联动进入 `interlocked`；浓度回落后需人工确认恢复（`restored`）。
- **报警去重与闭环**：同监测点同类型 20 分钟内重复报警合并（计数递增）；报警需工艺员确认并填写处置说明；超 1 小时未确认由每 5 分钟扫描任务自动升级。
- **横切关注点**：操作审计日志（联锁下发/恢复、报警处置、PM 标记、请求流水全部留痕）、报警升级扫描、全局错误处理与统一响应格式、请求 trace id。

## 二、架构与分层

```
web (原生 HTML/CSS/JS, go:embed)
        |
        | HTTP
        v
httpapi (路由 / handler / 统一响应 / 参数校验 / 分页)
        |
        v
middleware (requestID / security / access log / recover / audit)
        |
        v
service (用例编排：ingest / interlock / alert / sweepers / overview / audit / zone)
        |
        v
store (内存仓储 + JSON 原子持久化，按实体拆分 repository)
        |
        v
domain (实体 / 状态机 / 规则 / 枚举 / 错误码，无外部依赖)
        |
        v
config (环境变量覆盖 / Validate / ISO 阈值表 / 工艺阈值)
```

依赖方向严格单向：`main → httpapi → service → store → domain`，`middleware` 与 `httpapi` 可横向使用 `domain` 与 `store`，但 `domain` 不依赖任何外部世界。每一层职责独立、可按文件替换。

## 三、目录结构

```
.
├── go.mod / main.go          # 入口：config → store → service → httpapi → 内嵌 web
├── logging/
│   └── logging.go            # slog 全局初始化（LOG_LEVEL 控制级别）
├── config/
│   ├── config.go             # 配置聚合 + FromEnv + Validate + ISO 阈值表
│   └── process.go            # 工艺区（光刻/刻蚀/扩散）阈值再导出与校验
├── domain/                   # 领域层（实体 + 状态机 + 规则，无外部依赖）
│   ├── types.go              # 共享枚举：ZoneStatus / IsoClass / ProcessType / AlertType 等
│   ├── cleanzone.go          # 洁净区实体
│   ├── zone_state.go         # 状态机迁移表
│   ├── monitor.go            # 监测分区 + 阈值 + 设备
│   ├── sample.go             # 环境数据
│   ├── validity.go           # 数据有效性 + 无效占比
│   ├── iso.go                # ISO 判定（双粒径取较差）
│   ├── interlock.go          # 联锁日志
│   ├── alert.go              # 报警 + 去重 + 生命周期
│   ├── audit.go              # 审计记录
│   ├── process.go            # 工艺阈值
│   └── errors.go             # 领域错误码
├── store/                    # 仓储层（内存 + JSON 原子持久化）
│   ├── store.go              # 状态快照 / 写锁 / 序列 / 原子 Save / 损坏降级 Load
│   ├── cleanzone_store.go / monitor_store.go / sample_store.go
│   ├── interlock_store.go / alert_store.go / audit_store.go
│   └── json_persist.go       # JSON 编解码
├── service/                  # 服务层
│   ├── ingest_service.go     # 采集 + 有效性 + 等级判定 + 状态机 + 联锁 + 报警
│   ├── interlock_service.go  # 联锁下发（整区一致）/ 恢复
│   ├── alert_service.go      # 去重 / 确认 / 关闭 / 升级
│   ├── escalate_sweeper.go   # 每 5 分钟升级扫描
│   ├── zone_sweeper.go       # 超限 10 分钟自动联锁扫描
│   ├── audit_service.go / overview_service.go / zone_service.go
│   ├── bootstrap.go          # 演示数据种子（3 洁净区 / 6 监测点）
│   └── service.go            # 服务容器
├── httpapi/                  # API 层
│   ├── router.go / respond.go（统一响应）/ helpers.go
│   ├── zone_handler.go / sample_handler.go / interlock_handler.go
│   ├── alert_handler.go / overview_handler.go / health_handler.go
├── middleware/               # requestID / security / access log / recover / audit
│   ├── request_id.go / security.go / request_log.go
│   ├── error_handler.go / audit.go / response_recorder.go
├── web/                      # go:embed 内嵌原生前端
│   ├── index.html / app.js / api.js / style.css / dialog.js
│   ├── components/           # ZoneCard / ParticleTrend / InterlockTimeline
│   ├── hooks/                # useZones / useAlerts
│   └── pages/                # overview / zone_detail / alerts / interlocks / equipment
├── Dockerfile / .dockerignore / Makefile
└── runtime_smoke.json        # 冒烟契约（mode/start/ready_url/project_intro）
```

## 四、运行

```bash
# 默认监听 8080
go run .

# 指定端口（验证端口示例）
PORT=19004 go run .
```

启动后：

- 页面：<http://127.0.0.1:8080/>（端口按 `PORT` 调整）
- 健康检查：<http://127.0.0.1:8080/healthz>（另有 `/readyz`、`/api/healthz`，均返回 200）

> 说明：`runtime_smoke.json` 的 `ready_url` 为 `/healthz`；该路径现在由后端显式返回 200，不再依赖 SPA 回退。

## 五、Docker 部署

```bash
# 构建镜像（多阶段：golang:1.23-alpine → alpine:3.20，CGO_ENABLED=0，非 root）
docker build -t cleanroom-environment-monitor-service:latest .

# 运行（容器内尊重 PORT，默认 8080，内置 HEALTHCHECK）
docker run --rm -p 19004:8080 \
  -e PORT=8080 \
  -v "$PWD/data:/data" \
  -e DATA_FILE=/data/cleanroom_data.json \
  cleanroom-environment-monitor-service:latest

# 冒烟
curl -fsS http://127.0.0.1:19004/healthz
```

镜像特性：

- 构建阶段 `CGO_ENABLED=0`、`GOOS=linux`、`GOARCH=amd64`，产物为静态二进制；
- 运行阶段 `alpine:3.20`，创建非 root 用户 `app` 并以该用户运行；
- `EXPOSE 8080`，服务通过 `PORT` 环境变量覆盖监听端口；
- `HEALTHCHECK` 每 30 秒访问 `/healthz`，3 次失败判定不健康。

## 六、环境变量表

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | HTTP 监听端口 |
| `DATA_FILE` | `data/cleanroom_data.json` | JSON 持久化文件路径；设为空字符串禁用持久化 |
| `LOG_LEVEL` | `info` | 结构化日志级别：`debug` / `info` / `warn` / `error` |
| `SHUTDOWN_TIMEOUT` | `10s` | 优雅关闭最长等待时间 |
| `READ_HEADER_TIMEOUT` | `10s` | 读取请求头的超时 |
| `READ_TIMEOUT` | `30s` | 读取整个请求体的超时 |
| `WRITE_TIMEOUT` | `60s` | 写入响应的超时 |
| `IDLE_TIMEOUT` | `120s` | keep-alive 空闲连接超时 |
| `AUTO_INTERLOCK_AFTER` | `10m` | 持续偏高/超限多久后自动联锁 |
| `ALERT_ESCALATE_AFTER` | `1h` | 未确认报警多久后升级 |
| `ALERT_DEDUP_WINDOW` | `20m` | 同类报警去重合并窗口 |
| `INVALID_RATIO_THRESHOLD` | `0.30` | 无效数据占比阈值（0..1） |
| `INVALID_RATIO_WINDOW` | `50` | 计算无效占比使用的最近样本数 |
| `OVER_LIMIT_RATIO` | `1.5` | 判定 `over_limit` 的浓度/上限比 |
| `INTERLOCK_SWEEP_INTERVAL` | `1m` | 自动联锁扫描间隔 |
| `ESCALATE_SWEEP_INTERVAL` | `5m` | 报警升级扫描间隔 |
| `MAX_SAMPLES_PER_ZONE` | `2000` | 每个监测点保留的样本历史上限（0 表示不裁剪） |

时长变量均使用 Go `time.ParseDuration` 格式（例如 `10s`、`5m`、`1h`）。服务通过 `FromEnvStrict` 读取环境变量：解析失败或越界（空端口、非法日志级别、非正时长、越界阈值等）会在启动阶段直接拒绝，避免带病运行。

## 七、API 表

统一响应格式：

```json
{"code":0,"message":"ok","data":{...}}
{"code":404,"message":"clean zone \"x\" not found","error":"not_found","request_id":"req-..."}
```

分页列表接口（`GET /api/zones`、`GET /api/monitors`、`GET /api/alerts`、`GET /api/interlocks`、`GET /api/audit`、趋势接口）支持 `limit` / `offset`：

- `limit` 默认 `100`，上限 `1000`，`0` 表示不设上限（兼容旧行为）；
- `offset` 默认 `0`；
- 响应 `data` 仍为数组以兼容现有前端与测试；同时返回顶层 `total` 字段与 `X-Total-Count` / `X-Limit` / `X-Offset` 响应头。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz`、`/readyz`、`/api/healthz` | 健康检查（200 + 服务元数据） |
| GET | `/api/overview` | 总览聚合 |
| GET | `/api/zones` | 洁净区列表（分页） |
| POST | `/api/zones` | 创建洁净区 |
| GET | `/api/zones/{id}` | 洁净区详情 |
| GET | `/api/zones/{id}/samples` | 分区所有监测点采样趋势（分页） |
| GET | `/api/zones/{id}/interlocks` | 分区联锁记录（分页） |
| POST | `/api/zones/{id}/interlock` | 下发联锁动作（整区一致） |
| POST | `/api/zones/{id}/restore` | 恢复确认（必填 note） |
| GET | `/api/monitors` | 监测点列表（分页） |
| POST | `/api/monitors` | 创建监测点 |
| GET | `/api/monitors/{id}` | 监测点详情 |
| GET | `/api/monitors/{id}/samples` | 监测点采样趋势（分页） |
| POST | `/api/monitors/{id}/samples` | 环境数据上报（有效性 + 等级判定 + 联锁 + 报警） |
| POST | `/api/monitors/{id}/maintenance` | 标记设备 PM 维护 / 结束 |
| POST | `/api/monitors/{id}/calibration` | 更新标定到期时间（RFC3339） |
| GET | `/api/alerts` | 报警列表（`status`/`type`/`monitor_zone_id`/`clean_zone_id` 过滤 + 分页） |
| GET | `/api/alerts/{id}` | 报警详情 |
| POST | `/api/alerts/{id}/ack` | 确认报警（必填 `disposition`） |
| POST | `/api/alerts/{id}/escalate` | 手动升级报警 |
| GET | `/api/interlocks` | 联锁记录查询（分页） |
| GET | `/api/audit` | 审计日志（`limit`/`offset`/`action` 过滤） |

输入校验：JSON 载荷拒绝未知字段与尾随数据；采样数值必须为有限数，粒子计数非负；非法输入返回 400，不再 panic。

## 八、前端页面

| 路由 | 页面 | 使用的组件 / hooks |
|---|---|---|
| `/` | 洁净区总览 | `ZoneCard`、`useZones`、`useAlerts` |
| `/zones/{id}` | 分区详情（趋势 + 判定明细 + 联锁记录） | `ZoneCard`、`ParticleTrend`、`InterlockTimeline`、`useZones` |
| `/alerts` | 报警台（列表 + 确认 + 升级 + 趋势） | `useAlerts`、`ParticleTrend` |
| `/interlocks` | 联锁记录时间线 | `InterlockTimeline` |
| `/equipment` | 设备状态 + PM 维护标记 | 原生表格 + `POST maintenance` |

所有页面均真实消费后端接口（`web/api.js` 统一封装），非静态假页面。SPA 采用 pathname 路由，非 API 路径由服务端回退到 `index.html`。

## 九、共享枚举 / 常量（前后端各定义、保持一致）

| 枚举/常量 | Go 定义位置 | JS 定义位置 |
|---|---|---|
| 洁净区状态 `ZoneStatus`（normal/elevated/over_limit/interlocked/restored） | `domain/types.go`（`ZoneStatus*` 常量、`AllZoneStatuses`） | `web/components/zone_card.js`（`STATUS_LABEL`、`STATUS_CLASS`） |
| ISO 等级 `IsoClass`（iso5/iso6/iso7/iso8） | `domain/types.go`（`Iso*` 常量、`AllIsoClasses`）；阈值表 `config/config.go#ISO14644Limits` | `web/components/zone_card.js`（`ISO_LABEL`） |
| 报警类型 `AlertType`（particle/temp_humidity/pressure/data_quality） | `domain/types.go`（`Alert*` 常量、`AllAlertTypes`） | `web/pages/alerts.js`（`TYPE_LABEL`）、`web/pages/overview.js`（CSS class） |
| 联锁动作 `InterlockAction`（ffu_speed_up/fresh_air_increase/exhaust_increase） | `domain/types.go` + `domain/interlock.go#ActionsForLevel` | `web/components/interlock_timeline.js`（`ACTION_LABEL`） |
| 报警状态 `AlertStatus` / 级别 `AlertLevel` | `domain/types.go` | `web/pages/alerts.js`、`web/style.css` |
| 工艺类型 `ProcessType` | `domain/types.go` + `domain/process.go` | `web/components/zone_card.js`（`PROCESS_LABEL`） |

## 十、测试与质量门禁

```bash
go build ./...       # 编译
go vet ./...         # 静态检查
gofmt -l .           # 格式检查（应无输出）
go test ./...        # 单元 + 集成测试
go test -race ./...  # 并发竞态检查（必须全绿）
```

覆盖范围：状态机、ISO 判定、有效性、去重、整区联锁、升级扫描、持久化往返、HTTP 集成、中间件、请求 ID、panic 恢复。

## 十一、健康检查

- `GET /healthz` → 200：冒烟契约 `ready_url` 使用的就绪探针。
- `GET /readyz` → 200：同 `/healthz`，供 Kubernetes `readinessProbe` 使用。
- `GET /api/healthz` → 200：前端健康指示灯使用的 JSON 探针。

响应包含 `status`、`uptime_secs`、`zones`、`audit_events`、`time`。

## 十二、故障排查

| 现象 | 排查方向 |
|---|---|
| 启动报 `invalid config` | 检查 `LOG_LEVEL` 等环境变量取值与 `Validate()` 约束 |
| `load store` 失败 | 持久化文件目录不可写；若 JSON 损坏，服务会自动备份为 `<file>.bak` 并降级为空库启动（日志含 `degraded store start`） |
| `/healthz` 非 200 | 端口冲突（`PORT` 被占用）、启动失败，查看结构化日志中的 `http.request` 与错误日志 |
| 页面空白 | 确认 `web/` 被 `go:embed` 打包，`GET /` 返回 `index.html`；浏览器控制台查看 `/api/overview` 请求 |
| 数据每次重启丢失 | 确认 `DATA_FILE` 未设为空字符串，且运行账户对该路径有写权限 |
| 报警未升级 | 升级扫描默认每 5 分钟一次，可调 `ESCALATE_SWEEP_INTERVAL` / `ALERT_ESCALATE_AFTER` |

## 十三、可复现的核心链路（构造缺陷与验证用）

```bash
BASE=http://127.0.0.1:8080

# 1. 正常采样 → normal
curl -sS -X POST "$BASE/api/monitors/monitor_a1/samples" -H 'Content-Type: application/json' \
  -d '{"count_0_3um":30000,"count_0_5um":8000,"temperature":21,"humidity":45,"pressure_diff":20}'

# 2. 偏高采样（1.0~1.5×，monitor_a1 上限 80000/28000）→ elevated + particle 报警
curl -sS -X POST "$BASE/api/monitors/monitor_a1/samples" -H 'Content-Type: application/json' \
  -d '{"count_0_3um":90000,"count_0_5um":30000,"temperature":21,"humidity":45,"pressure_diff":20}'

# 3. 超限采样（≥1.5×）→ 整区（PA-A 的 zone_a1/zone_b1）联锁通风 + 联锁日志
curl -sS -X POST "$BASE/api/monitors/monitor_a1/samples" -H 'Content-Type: application/json' \
  -d '{"count_0_3um":140000,"count_0_5um":50000,"temperature":21,"humidity":45,"pressure_diff":20}'

# 4. 恢复确认
curl -sS -X POST "$BASE/api/zones/zone_a1/restore" -H 'Content-Type: application/json' \
  -d '{"operator":"eng_li","note":"更换过滤器并复测"}'

# 5. PM 维护 → 数据无效 → 数据可信度报警
curl -sS -X POST "$BASE/api/monitors/monitor_a1/maintenance" -H 'Content-Type: application/json' \
  -d '{"in_maintenance":true,"note":"PM 校准"}'
curl -sS -X POST "$BASE/api/monitors/monitor_a1/samples" -H 'Content-Type: application/json' \
  -d '{"count_0_3um":30000,"count_0_5um":8000,"temperature":21,"humidity":45,"pressure_diff":20}'

# 6. 报警确认（必填处置说明）
curl -sS -X POST "$BASE/api/alerts/<id>/ack" -H 'Content-Type: application/json' \
  -d '{"operator":"eng_wang","disposition":"擦拭传感器镜片"}'
```
