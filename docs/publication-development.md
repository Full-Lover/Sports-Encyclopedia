# 本地发布流程（预览阶段）

当前网站默认仍使用仓库内的不可变预览快照，不需要 MySQL。设置 `SPORTS_DB_DSN` 后，`web` 改从 MySQL 的当前指针读取已发布快照；此时数据库未迁移或尚无已发布快照，网站不会悄悄退回静态样例，而是返回不可用。

本阶段的 `publish-preview` 仅把内置 `preview-0007` 内容作为开发测试候选发布，用来验证发布机制。它不是未来的外部数据同步，也不代替 `AtlasRegistry` 的资料核验。

在 PowerShell 中给当前终端设置 MySQL DSN（密码不要写入仓库或聊天），再依次运行：

```powershell
$env:SPORTS_DB_DSN = '<本机私有 DSN，数据库需预先创建>'
go run ./cmd/sports-encyclopedia migrate
go run ./cmd/sports-encyclopedia publish-preview
npm --prefix web run build
go run ./cmd/sports-encyclopedia web
```

DSN 使用 Go MySQL 驱动格式，例如 `用户名:密码@tcp(127.0.0.1:3306)/数据库名`。程序强制使用 `parseTime=true` 和 UTC 解析数据库时间。`migrate` 现在依次添加 `PublishedAtlas` 和 `AtlasRegistry` 各自拥有的表；两个 Module 的 `down.sql` 都只供人工审查，程序不会自动执行会删除数据的回退迁移。

真实数据库集成测试只在设置 `SPORTS_TEST_MYSQL_DSN` 后运行，并且数据库名称必须含有 `test`；测试会写入快照和指针，请使用专用的可丢弃测试库：

```powershell
$env:SPORTS_TEST_MYSQL_DSN = '<专用测试库的私有 DSN>'
go test -count=1 -run TestMySQLPublicationLifecycle -v ./internal/publishedatlas
go test -count=1 ./internal/atlasregistry
```

发布过程先把候选写为不可见，再在同一个 InnoDB 事务中校验租约、发布门、当前基线和候选，记录收据并切换当前指针。事务失败时当前指针不变；过期租约先检查旧运行收据，再决定恢复还是让新运行接管。读侧只接受 `PUBLISHED` 快照。

`AtlasRegistry` 现已提供事实/媒体暂存、运行封存与废弃、四联盟规则、来源核验、冲突与失败保留、基线编译、旧 slug 保护及逐文件媒体授权。它将来源批次和编译后的规范化内容作为不可变 JSON 聚合保存在 Registry 自有表中，并用独立的策略、授权决定和 slug 表维护需要跨基线约束的记录；这不是按每种领域实体各建一张表的物理映射。媒体仅有来源声明还不能展示：受信任的本地审核必须通过 `InstallMediaRightsDecision` 为精确文件修订选择权利依据，未知或撤销的决定默认回退为球队缩写/图片占位。

当前 `publish-preview` 仍只发布内置样例，不调用 Registry。缺少的是下一 Module `AtlasRefresh` 的来源 Adapter、编排和 Registry→PublishedAtlas DTO 映射，以及正式页面/搜索文档；因此 Registry 的实现不会直接改变当前网页。
