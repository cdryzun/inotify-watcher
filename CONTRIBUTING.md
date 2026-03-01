# Contributing to TrueNAS Artifact Inotify Hook

感谢你考虑为本项目做出贡献！🎉

## 📋 目录

- [行为准则](#行为准则)
- [如何贡献](#如何贡献)
- [开发流程](#开发流程)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [测试要求](#测试要求)
- [Pull Request 流程](#pull-request-流程)

## 行为准则

本项目采用 [Contributor Covenant](CODE_OF_CONDUCT.md) 行为准则。参与此项目即表示你同意遵守其条款。

## 如何贡献

### 报告 Bug

如果你发现了 bug，请通过 [GitHub Issues](../../issues) 提交报告。提交前请：

1. 搜索现有 issues，确认该问题尚未被报告
2. 使用 Issue 模板填写详细信息
3. 提供可复现的步骤、预期行为和实际行为

### 建议新功能

欢迎提出新功能建议！请：

1. 通过 Issues 提交功能请求
2. 详细描述功能和使用场景
3. 说明为什么这个功能对项目有价值

### 提交代码

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 编写代码和测试
4. 确保所有测试通过 (`go test ./...`)
5. 提交变更 (`git commit -m 'feat: add amazing feature'`)
6. 推送到分支 (`git push origin feature/amazing-feature`)
7. 提交 Pull Request

## 开发流程

### 环境要求

- Go 1.21+
- Task (可选，用于构建任务)

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/YOUR_USERNAME/truenas-artifact-inotify-hook.git
cd truenas-artifact-inotify-hook

# 安装依赖
go mod download

# 运行测试
go test -v -race -cover ./...

# 构建
task build

# 运行
./dist/truenas-artifact-inotify-hook --help
```

## 代码规范

### Go 代码规范

- 遵循 [Effective Go](https://golang.org/doc/effective_go) 指南
- 使用 `gofmt` 格式化代码
- 使用 `go vet` 检查代码
- 使用 `golangci-lint` 进行综合检查

### 代码质量标准

- **测试覆盖率**: 新代码必须包含单元测试，覆盖率不低于 80%
- **文档**: 导出的函数和类型必须有注释
- **错误处理**: 所有错误必须正确处理，不得忽略
- **命名**: 使用清晰、描述性的名称

### 示例

```go
// Add adds a path to the watch list.
// If recursive is enabled, all subdirectories are also watched.
func (w *Watcher) Add(path string) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("failed to get absolute path: %w", err)
    }
    // ...
}
```

## 提交规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

### 提交格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档变更
- `style`: 代码格式（不影响代码运行的变动）
- `refactor`: 重构（既不是新增功能，也不是修改 bug 的代码变动）
- `test`: 增加测试
- `chore`: 构建过程或辅助工具的变动
- `perf`: 性能优化

### 示例

```bash
feat: add recursive directory watching support
fix: prevent duplicate events on rapid file changes
docs: update README with usage examples
test: add integration tests for watcher
refactor: simplify event parsing logic
```

## 测试要求

### 运行测试

```bash
# 运行所有测试
go test -v ./...

# 运行带覆盖率的测试
go test -cover ./...

# 运行竞态检测
go test -race ./...

# 运行特定测试
go test -v -run TestAdd ./...
```

### 测试标准

- 所有新功能必须有单元测试
- Bug 修复必须包含回归测试
- 测试覆盖率不低于 80%
- 使用 table-driven tests 模式
- 集成测试标记为 `//go:build integration`

## Pull Request 流程

### 提交前检查清单

- [ ] 代码通过 `gofmt` 格式化
- [ ] 代码通过 `go vet` 检查
- [ ] 代码通过 `golangci-lint run` 检查
- [ ] 所有测试通过 (`go test -race -cover ./...`)
- [ ] 新代码有足够的测试覆盖
- [ ] 文档已更新（如需要）
- [ ] 提交信息遵循 Conventional Commits 规范

### PR 标题

PR 标题应遵循与提交信息相同的格式：

```
feat: add support for custom event filters
fix: resolve memory leak in watcher loop
docs: add deployment guide for TrueNAS SCALE
```

### 代码审查

- 所有 PR 需要至少一位维护者审查
- 审查者会检查代码质量、测试覆盖和文档
- 请及时响应审查意见
- 保持讨论专业和建设性

### 合并要求

- ✅ 所有 CI 检查通过
- ✅ 至少一位维护者批准
- ✅ 没有未解决的讨论
- ✅ 分支与主分支同步

## 获取帮助

- 💬 [GitHub Discussions](../../discussions) - 一般性问题讨论
- 🐛 [GitHub Issues](../../issues) - Bug 报告和功能请求
- 📧 邮件: dev@your-domain.com

## 许可证

通过提交代码，你同意你的贡献将根据 [MIT License](LICENSE) 进行许可。

---

再次感谢你的贡献！❤️
