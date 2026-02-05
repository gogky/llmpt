# 代码重构说明

## 📌 重构目标

将代码从 `pkg/` 目录重构到 `internal/` 目录，遵循 Go 项目最佳实践。

## 🔄 重构内容

### 目录结构变化

**重构前:**
```
llmpt/
├── pkg/
│   ├── config/
│   ├── database/
│   └── models/
```

**重构后:**
```
llmpt/
├── internal/          # 👈 项目内部代码
│   ├── config/
│   ├── database/
│   └── models/
```

### 文件移动清单

| 旧路径 | 新路径 |
|--------|--------|
| `pkg/config/config.go` | `internal/config/config.go` |
| `pkg/database/db.go` | `internal/database/db.go` |
| `pkg/database/mongodb.go` | `internal/database/mongodb.go` |
| `pkg/database/redis.go` | `internal/database/redis.go` |
| `pkg/models/torrent.go` | `internal/models/torrent.go` |

### Import 路径更新

所有 import 语句已从 `llmpt/pkg/*` 更新为 `llmpt/internal/*`：

**重构前:**
```go
import (
    "llmpt/pkg/config"
    "llmpt/pkg/database"
    "llmpt/pkg/models"
)
```

**重构后:**
```go
import (
    "llmpt/internal/config"
    "llmpt/internal/database"
    "llmpt/internal/models"
)
```

## ✅ 为什么使用 `internal/`？

### 1. **Go 编译器强制保护**
- `internal/` 包只能被**同一模块**内的代码导入
- 防止外部项目意外依赖你的内部实现
- 提供更强的封装性

### 2. **符合 Go 项目规范**
- Go 标准库和大型开源项目（如 Kubernetes）都遵循这个模式
- `pkg/` 通常用于可以被其他项目导入的公共库
- `internal/` 用于应用程序的内部实现

### 3. **明确项目定位**
- llmpt 是一个**应用程序**，不是供他人导入的库
- 数据库连接代码包含业务逻辑，不应该被外部使用
- 更好的代码组织和维护

## 🎯 影响范围

### ✅ 已更新的文件
- [x] `internal/config/config.go`
- [x] `internal/database/db.go`
- [x] `internal/database/mongodb.go`
- [x] `internal/database/redis.go`
- [x] `internal/models/torrent.go`
- [x] `cmd/test-db/main.go`
- [x] `DATABASE_SETUP.md`

### 🧪 测试结果

重构后所有测试均通过：

```
✓ Successfully connected to MongoDB
✓ Successfully connected to Redis
✓ MongoDB indexes created successfully
✓ All MongoDB operations tested
✓ All Redis operations tested
✓ 所有测试完成!
```

## 📋 后续开发注意事项

1. **新增代码位置**
   - 业务逻辑代码应放在 `internal/` 下
   - 例如：`internal/api/`, `internal/tracker/`, `internal/service/`

2. **通用工具代码**
   - 如果需要创建真正可复用的通用工具，可以放在 `pkg/` 下
   - 但要确保这些工具没有业务逻辑耦合

3. **Import 路径规范**
   ```go
   // 内部代码
   import "llmpt/internal/config"
   import "llmpt/internal/database"
   
   // 第三方库
   import "github.com/redis/go-redis/v9"
   ```

## 🔍 验证方法

重构完成后，可以通过以下方式验证：

```bash
# 1. 运行测试
go run cmd/test-db/main.go

# 2. 编译检查
go build ./...

# 3. 模块整理
go mod tidy
```

## 📚 参考资料

- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Go Internal Packages](https://docs.google.com/document/d/1e8kOo3r51b2BWtTs_1uADIA5djfXhPT36s6eHVRIvaU/edit)
- [Effective Go](https://go.dev/doc/effective_go)

---

**重构日期:** 2026-02-01  
**测试状态:** ✅ 通过  
**向后兼容:** ✅ 无影响（项目初期重构）
