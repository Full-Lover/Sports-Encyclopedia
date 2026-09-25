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

DSN 使用 Go MySQL 驱动格式，例如 `用户名:密码@tcp(127.0.0.1:3306)/数据库名`。程序强制使用 `parseTime=true` 和 UTC 解析数据库时间。迁移仅添加 `PublishedAtlas` 所拥有的表；`migrations/0001_publication.down.sql` 是人工审查用的回退文件，程序不会自动执行会删除发布数据的 down 迁移。

真实数据库集成测试只在设置 `SPORTS_TEST_MYSQL_DSN` 后运行，并且数据库名称必须含有 `test`；测试会写入快照和指针，请使用专用的可丢弃测试库：

```powershell
$env:SPORTS_TEST_MYSQL_DSN = '<专用测试库的私有 DSN>'
go test -count=1 -run TestMySQLPublicationLifecycle -v ./internal/publishedatlas
```

发布过程先把候选写为不可见，再在同一个 InnoDB 事务中校验租约、发布门、当前基线和候选，记录收据并切换当前指针。事务失败时当前指针不变；过期租约先检查旧运行收据，再决定恢复还是让新运行接管。读侧只接受 `PUBLISHED` 快照。当前仍缺少自动刷新、Registry 编译和搜索文档，因此这套机制目前只承载预览地图与由其派生的球队页。
