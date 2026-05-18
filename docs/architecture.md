# bidking 架构分层规范

本文档是后续实现必须遵守的基础架构约定。它基于当前仓库状态、多人实时游戏常见实践、Unity 官方 Assembly Definition/ScriptableObject 工作流、Go 官方 `internal` 包边界，以及服务端权威的联网游戏设计原则整理。

当前项目仍处于原型早期：

- Unity 客户端已有 `App`、`Game`、`Config`、`Networking` 初始目录。
- Go 服务端已有 `cmd/gameserver`、`internal/app`、`internal/httpapi`、`internal/realtime`、`internal/game`、`internal/store`。
- `shared/config` 和 `shared/protocol` 已经作为前后端共享规则与协议入口。

后续新增功能以本文档为准。除非有明确收益，不为“架构整洁”做大规模搬家。

## 1. 架构目标

### 1.1 核心原则

- 服务端权威：房间状态、出价合法性、胜负、金币扣减、藏品归属和结算结果只由 Go 服务端决定。
- 客户端表现：Unity 只负责展示、输入、本地动效、音效、临时 UI 状态和乐观反馈，不能生成最终游戏结果。
- 配置共享：规则、藏品、地图、场所、道具、协议枚举优先从 `shared/` 读取或生成，避免两端各写一份。
- 分层单向依赖：外层可以依赖内层，内层不能依赖外层。
- 领域可测试：核心竞拍规则不依赖 Unity、HTTP、WebSocket、数据库、Redis 或真实时间。
- 渐进实现：MVP 先跑通核心竞拍闭环，架构允许替换实现，但不提前引入过重框架。

### 1.2 总体分层

```text
Unity Client
  Presentation       UI、场景、Prefab、MonoBehaviour View
  Application        Presenter / Controller / UseCase Facade
  Domain             客户端只读模型、枚举、状态快照、展示推导
  Infrastructure     WebSocket、配置加载、资源加载、本地存储、SDK 适配
  App                启动、依赖组装、生命周期、场景切换

WebSocket JSON Protocol
  shared/protocol/messages.json
  shared/config/*.json

Go Server
  Adapter            HTTP、WebSocket、DB、Redis、日志、定时器适配
  Application        用例编排：创建房间、加入房间、准备、出价、结算
  Domain             房间状态机、竞拍规则、仓库生成、估值、结算
  Infrastructure     PostgreSQL/MySQL、Redis、配置源、外部服务实现
  App                依赖组装、进程启动、优雅关闭
```

### 1.3 依赖方向

```text
Unity:
Presentation -> Application -> Domain
Presentation -> Infrastructure 接口外观
Application  -> Domain
Application  -> Infrastructure 接口
Infrastructure -> Domain
App -> all

Go:
cmd/gameserver -> internal/app
internal/app -> adapters + application + infrastructure
adapters -> application
application -> domain + ports
infrastructure -> ports + domain value objects
domain -> standard library only
```

任何反向依赖都要先停下来重看设计。

## 2. 共享层 shared

`shared/` 是跨 Unity 和 Go 的契约源，不直接包含运行时代码。

```text
shared/
├── config/
│   ├── game_rules.json
│   ├── collectibles.json
│   ├── items.json
│   ├── venues.json
│   ├── maps.json
│   └── roles.json
└── protocol/
    └── messages.json
```

### 2.1 职责

- `config`：玩法配置、经济数值、内容表、地图和场所定义。
- `protocol`：消息类型、方向、payload schema 的源头。
- 文档口径：玩法语义以 `docs/game-rules.md` 为准；实现字段以 `shared/` 为准。两者冲突时先修文档和配置，再写代码。

### 2.2 约束

- 配置文件字段使用稳定 ID，不依赖展示名。
- 服务端启动时校验配置完整性，校验失败直接启动失败。
- Unity 可以把 JSON 导入成 ScriptableObject 资产，但运行时逻辑仍要能追溯到 shared ID。
- 协议 payload 必须可版本化，新增字段保持向后兼容；删除/改名需要迁移说明。

## 3. Unity 客户端分层

Unity 采用轻量 MV* + Service/Port 模式。不要一开始引入大型 DI 框架；使用 Bootstrap 手动组装依赖，等复杂度真正上来再考虑容器。

### 3.1 推荐目录

当前目录已有 `App`、`Config`、`Game`、`Networking`，后续按下面结构演进：

```text
client/Assets/_Project/
├── Scripts/
│   ├── App/
│   │   ├── AppBootstrap.cs
│   │   ├── AppLifetime.cs
│   │   └── SceneFlow.cs
│   ├── Domain/
│   │   ├── Models/
│   │   ├── Rules/
│   │   └── Protocol/
│   ├── Application/
│   │   ├── Ports/
│   │   ├── Sessions/
│   │   └── Presenters/
│   ├── Infrastructure/
│   │   ├── Networking/
│   │   ├── Config/
│   │   ├── Storage/
│   │   └── Time/
│   ├── Presentation/
│   │   ├── Views/
│   │   ├── ViewModels/
│   │   ├── Widgets/
│   │   └── Screens/
│   └── Tests/
├── ScriptableObjects/
│   ├── Config/
│   └── EventChannels/
├── Prefabs/
├── Scenes/
├── UI/
├── Art/
├── Audio/
└── Materials/
```

短期内可以保留已有目录，新增代码优先按新分层放置。已有文件迁移建议：

| 当前文件 | 目标层 | 后续位置 |
|---|---|---|
| `App/AppBootstrap.cs` | App | `Scripts/App/AppBootstrap.cs` |
| `Game/AuctionModels.cs` | Domain | `Scripts/Domain/Models/RoomSnapshot.cs` 等拆分 |
| `Game/AuctionSession.cs` | Application | `Scripts/Application/Sessions/AuctionSession.cs` |
| `Networking/RealtimeClient.cs` | Infrastructure | `Scripts/Infrastructure/Networking/RealtimeClient.cs` |
| `Config/ItemCatalog.cs` | Infrastructure/Config Asset | `Scripts/Infrastructure/Config/ItemCatalog.cs` 或生成器 |

### 3.2 Unity 层职责

| 层 | 可以做 | 不能做 |
|---|---|---|
| App | 启动、依赖组装、跨场景对象、全局生命周期 | 写具体竞拍规则 |
| Presentation | MonoBehaviour、UI 绑定、动画触发、输入采集、屏幕切换 | 直接改房间结果、直接拼 WebSocket payload |
| Application | 玩家意图编排、Presenter、Session、调用端口接口、处理快照 | 依赖 Unity UI 组件、依赖具体 WebSocket 类 |
| Domain | 快照模型、枚举、只读展示推导、客户端校验提示 | 调 Unity API、发网络、读磁盘、决定最终输赢 |
| Infrastructure | WebSocket、JSON、配置加载、本地存储、SDK | 写 UI 逻辑、写核心竞拍算法 |

### 3.3 Assembly Definition 规划

Unity 官方建议用 Assembly Definition 管理脚本程序集边界和编译依赖。后续从一个 `BidKing.Client.asmdef` 拆成：

```text
BidKing.Client.Domain
BidKing.Client.Application      references Domain
BidKing.Client.Infrastructure   references Domain, Application
BidKing.Client.Presentation     references Domain, Application
BidKing.Client.App              references all
BidKing.Client.Tests            references tested assemblies
```

约束：

- `Domain` 的 asmdef 不引用 UnityEngine，除非只是必要的序列化属性；优先保持纯 C#。
- `Application` 不引用 `Presentation`。
- `Infrastructure` 不引用 `Presentation`。
- `Presentation` 可以引用 `Application` 的接口或 Presenter，但不能 new 具体网络实现。
- `AppBootstrap` 是少数允许知道所有实现类的地方。

### 3.4 客户端数据流

```text
玩家点击出价按钮
  -> AuctionView 采集输入
  -> AuctionPresenter.PlaceBid(amount)
  -> IAuctionGateway.PlaceBidAsync(amount)
  -> RealtimeClient 发送 auction.bid
  -> Go 服务端校验
  -> 服务端广播 room.snapshot / auction.bid_accepted
  -> RealtimeClient 解析消息
  -> AuctionSession 更新当前快照
  -> Presenter/ViewModel 通知 View 刷新
```

规则：

- View 只发出“玩家意图”，不直接改 `RoomSnapshot`。
- Presenter 可以做本地输入校验，例如金额为空、不是数字、超过本地已知金币；但最终成功/失败以服务端消息为准。
- UI 动画可以先做“提交中”反馈，不能提前展示“获胜”。
- 客户端计时器只用于显示倒计时；出价窗口是否结束由服务端快照决定。

### 3.5 客户端状态模型

客户端状态分三类：

| 类型 | 示例 | 来源 |
|---|---|---|
| 权威状态 | 房间阶段、玩家金币、出价结果、藏品归属 | 服务端快照 |
| 派生展示状态 | 当前玩家是否可点击出价、收益展示、排序 | 本地从快照计算 |
| 临时 UI 状态 | 输入框内容、按钮 loading、动画播放进度 | Unity 本地 |

权威状态只能由网络消息更新。

### 3.6 ScriptableObject 使用边界

ScriptableObject 适合：

- 内容配置资产，如导入后的藏品目录、UI 主题、音效表。
- Event Channel，如 `RoomSnapshotUpdatedChannel`、`RoundSettledChannel`。
- 编辑器可配置的服务入口，如环境配置。

ScriptableObject 不适合：

- 存放服务端权威房间状态。
- 做复杂运行时状态机的唯一载体。
- 藏跨局持久数据，除非明确是只读配置。

Event Channel 可以解耦 UI 和 Presenter，但不要让事件链变成隐形业务流程。跨层核心流程仍由 Application 层显式编排。

## 4. Go 服务端分层

Go 服务端采用 Clean Architecture 风格，但使用 Go 的包习惯保持简单。当前 `internal/game` 作为领域层保留，不强制改名为 `domain`。

### 4.1 推荐目录

```text
server/
├── cmd/
│   └── gameserver/
│       └── main.go
├── internal/
│   ├── app/
│   │   └── app.go
│   ├── config/
│   │   └── config.go
│   ├── game/                    # Domain: 纯游戏规则
│   │   ├── room.go
│   │   ├── phase.go
│   │   ├── player.go
│   │   ├── warehouse.go
│   │   ├── bidding.go
│   │   ├── settlement.go
│   │   └── errors.go
│   ├── application/             # Use cases
│   │   ├── ports.go
│   │   ├── room_service.go
│   │   ├── command_handlers.go
│   │   └── snapshots.go
│   ├── realtime/                # WebSocket adapter
│   │   ├── hub.go
│   │   ├── client.go
│   │   ├── message_router.go
│   │   └── envelope.go
│   ├── httpapi/                 # HTTP adapter
│   │   └── router.go
│   ├── store/                   # Repository ports + implementations
│   │   ├── ports.go
│   │   ├── memory/
│   │   └── postgres/
│   └── clock/
│       ├── clock.go
│       └── system.go
├── configs/
└── go.mod
```

Go 官方 `internal` 目录会限制外部模块 import，适合放服务端私有代码。`cmd/gameserver` 只做进程入口。

### 4.2 Go 层职责

| 层/包 | 职责 | 依赖 |
|---|---|---|
| `cmd/gameserver` | main、退出码、日志兜底 | `internal/app` |
| `internal/app` | 读取配置、组装依赖、启动 HTTP server、优雅关闭 | 所有外层实现 |
| `internal/config` | 环境变量、配置路径、shared JSON 加载入口 | 标准库 |
| `internal/game` | 房间状态机、竞拍、平局、结算、仓库生成、领域错误 | 标准库 |
| `internal/application` | 用例编排、事务边界、端口接口、快照转换 | `game` |
| `internal/realtime` | WebSocket 连接、读写循环、消息路由、广播 | `application` |
| `internal/httpapi` | HTTP 路由、healthz、ws endpoint | `realtime` |
| `internal/store` | 仓储接口和实现 | `application` ports |
| `internal/clock` | 可替换时间源 | 标准库 |

### 4.3 领域层规则

`internal/game` 是最重要的层。

必须满足：

- 不 import `net/http`、WebSocket、SQL、Redis、日志框架。
- 不读取环境变量，不读写文件。
- 不使用真实系统时间；需要时间时通过参数传入或由 Application 层注入。
- 所有状态变化返回明确错误，例如 `ErrRoomFull`、`ErrInvalidPhase`、`ErrBidTooLow`。
- 对外暴露行为方法，避免外层直接改字段。

推荐形态：

```go
type Room struct {
	id      string
	phase   Phase
	players map[PlayerID]Player
	rounds  []Round
}

func (r *Room) Join(player Player) error
func (r *Room) SetReady(playerID PlayerID, ready bool) error
func (r *Room) Start(now time.Time) error
func (r *Room) PlaceBid(playerID PlayerID, amount Coins, now time.Time) error
func (r *Room) SettleRound(now time.Time) (RoundResult, error)
func (r *Room) SnapshotFor(playerID PlayerID) RoomSnapshot
```

字段先私有化，再通过方法维护不变量。当前 `Room` 字段还是 public，后续实现状态机时应收敛。

### 4.4 Application 层规则

Application 层负责把外部请求转成领域行为：

- 校验玩家身份和房间归属。
- 找到目标房间。
- 调用 `game.Room` 方法。
- 持久化必要结果。
- 生成广播事件和快照。
- 处理超时、断线、重连和幂等。

Application 层定义端口接口，外层实现：

```go
type RoomRepository interface {
	Get(ctx context.Context, roomID string) (*game.Room, error)
	Save(ctx context.Context, room *game.Room) error
}

type Broadcaster interface {
	ToRoom(ctx context.Context, roomID string, msg Message) error
	ToPlayer(ctx context.Context, playerID string, msg Message) error
}
```

不要让 `realtime.Hub` 直接修改 `game.Room`。Hub 只能调用 Application 用例。

### 4.5 房间并发模型

MVP 推荐：每个房间一个串行执行单元。

实现可以是：

- 内存期：`RoomActor`，一个 room goroutine + command channel。
- 持久化期：Application 层用 room-level lock 或 Redis lock 保证同一房间命令串行。

规则：

- 同一房间内命令串行处理。
- 不同房间可以并行。
- WebSocket 每个连接有独立读写循环，写入使用 buffered channel，避免慢客户端阻塞房间逻辑。
- 房间状态变更完成后再广播快照。

### 4.6 协议层规则

统一消息信封：

```json
{
  "type": "auction.bid",
  "requestId": "client-generated-id",
  "payload": {},
  "sentAt": 1700000000000
}
```

服务端响应可以带 `requestId`，便于客户端把 loading 状态和结果对应起来。

核心消息：

| 方向 | type | 说明 |
|---|---|---|
| C -> S | `auth.guest` | 游客登录 |
| C -> S | `room.create` | 创建房间 |
| C -> S | `room.join` | 加入房间 |
| C -> S | `room.ready` | 准备/取消准备 |
| C -> S | `auction.bid` | 暗拍出价 |
| C -> S | `room.leave` | 离开房间 |
| S -> C | `auth.accepted` | 登录成功 |
| S -> C | `room.snapshot` | 权威房间快照 |
| S -> C | `auction.bid_accepted` | 出价被接受 |
| S -> C | `auction.bid_rejected` | 出价被拒绝 |
| S -> C | `auction.round_settled` | 本轮结算事件 |
| S -> C | `error` | 通用错误 |

原则：

- `room.snapshot` 是最终真相。
- 事件消息用于表现和提示，不能替代快照。
- 客户端重连后先请求/接收最新快照，再恢复 UI。

## 5. 配置和内容管线

### 5.1 加载顺序

服务端启动：

```text
读取 env -> 定位 shared/config -> 解析 JSON -> schema 校验 -> 交叉引用校验 -> 构建只读规则表 -> 启动 HTTP/WebSocket
```

Unity 启动：

```text
读取本地环境配置 -> 加载内置 shared 配置或远端配置 -> 构建只读 Catalog -> 连接服务端 -> 接收快照
```

### 5.2 ID 规范

- 玩家：`player_xxx`
- 房间：`room_xxx`
- 藏品：`collectible_xxx`
- 道具：`item_xxx`
- 地图：`map_xxx`
- 场所：`venue_xxx`

展示名可以改，ID 不轻易改。

## 6. 测试策略

| 层 | 测试类型 | 目标 |
|---|---|---|
| Go `internal/game` | 单元测试 | 覆盖出价、平局、流拍、结算、金币扣减 |
| Go `internal/application` | 单元测试 | 用 fake repo/broadcaster 验证用例编排 |
| Go `realtime/httpapi` | 集成测试 | healthz、ws 握手、消息路由、错误返回 |
| Unity Domain/Application | EditMode | 快照解析、Presenter 状态、输入校验 |
| Unity Presentation | PlayMode 冒烟 | 场景能打开，主要按钮能触发 Presenter |
| E2E | 手动或脚本 | 两个客户端完成一局 MVP |

新增核心规则必须先补 Go `internal/game` 单测。

## 7. 实现顺序

### Phase 1: 本地核心闭环

- Unity 完成主要界面和本地假数据快照。
- Go `internal/game` 完成房间状态机和竞拍结算单测。
- `shared/config` 补齐 schema 和校验脚本。

### Phase 2: 服务端用例和 WebSocket

- 新增 `internal/application`。
- `realtime.Hub` 只做连接和路由。
- 跑通 `auth.guest -> room.create -> room.ready -> auction.bid -> room.snapshot`。

### Phase 3: Unity 真连接

- `RealtimeClient` 接入真实 WebSocket。
- `AuctionSession` 维护权威快照。
- Presenter/ViewModel 驱动 UI。

### Phase 4: 持久化和重连

- 实现 store。
- 支持断线重连、房间恢复、结算记录。

### Phase 5: 内容扩展

- 增加道具、地图、场所、长期经济。
- 保持规则配置驱动，避免硬编码到 UI 或 adapter。

## 8. 代码放置决策表

| 要新增的东西 | 放哪里 |
|---|---|
| “出价必须大于 0 且不能超过金币” | Go `internal/game`，Unity 只做提示性校验 |
| “按钮点击后禁用 1 秒” | Unity `Presentation` |
| “玩家点击出价后发送消息” | Unity `Application` 调端口，`Infrastructure` 实现 WebSocket |
| “房间里所有人准备后开局” | Go `internal/game` + `application` |
| “WebSocket 消息 JSON 解析” | Go `internal/realtime`，Unity `Infrastructure/Networking` |
| “藏品稀有度比例” | `shared/config` + Go config loader |
| “结算动画逐个展示藏品” | Unity `Presentation`，数据来自服务端事件/快照 |
| “数据库保存对局结果” | Go `internal/store` 实现，接口由 Application 持有 |
| “NPC 出价策略” | Go Domain/Application；本地 Demo 可在 Unity Application 做 fake，但联机版以服务端为准 |

## 9. 禁止事项

- 禁止 Unity View 直接 new 网络客户端并发送协议。
- 禁止 Unity 客户端决定最终胜者、金币、藏品归属。
- 禁止 Go `internal/game` import WebSocket、HTTP、SQL、Redis。
- 禁止 adapter 直接改 `Room` 字段绕过领域方法。
- 禁止在多处硬编码同一个规则数值；规则优先放 `shared/config`。
- 禁止为了单个功能跨层传递巨型全局对象。
- 禁止把临时 UI 状态写进服务端权威快照。

## 10. 参考实践

- Unity Manual: Introduction to assemblies in Unity，用于管理脚本程序集和依赖边界。https://docs.unity.cn/6000.1/Documentation/Manual/assembly-definitions-intro.html
- Unity Manual: ScriptableObject，用于保存可复用数据资产。https://docs.unity3d.com/Manual/class-ScriptableObject.html
- Go Modules: Organizing a Go module，包含 `cmd`、`internal` 等布局建议。https://go.dev/doc/modules/layout
- Go Specification: Internal packages import rule。https://go.dev/ref/spec#Import_declarations
- Go Blog: Pipelines and cancellation，可参考 goroutine/channel 的取消和收尾方式。https://go.dev/blog/pipelines
