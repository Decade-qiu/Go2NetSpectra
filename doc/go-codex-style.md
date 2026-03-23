# Go Codex 编码规范

本文档是为 Codex 和本仓库开发者整理的 Go 编码规范汇编，目标是把仓库内高频会遇到的 Go 风格、API 设计、错误处理、并发、测试与模块版本规则收敛到一份可执行手册中。

## 文档目标

- 作为根目录 `AGENTS.md` 的详细补充，便于 Codex 和开发者查阅。
- 尽量覆盖 Go 官方与准官方文档中最稳定、最可执行的规则。
- 在官方规则之外，只补充少量适合本仓库长期维护的执行建议。

## 规则级别

- `MUST`：默认必须遵守，除非有明确且记录在案的例外。
- `SHOULD`：强烈建议遵守；如不遵守，应能说明理由。
- `AVOID`：默认避免；只有在收益明显高于维护成本时才使用。

## 1. 总体设计原则

- `MUST`：优先保证代码清晰度。读者必须能快速看懂代码在做什么，以及为什么这样做。
- `MUST`：优先选择最简单、最标准、最容易维护的实现方式。
- `SHOULD`：复杂性只在性能、兼容性、并发模型或 API 约束确实需要时引入。
- `SHOULD`：如果代码因为性能或协议原因变复杂，需要配套注释、文档或测试说明关键约束。
- `SHOULD`：优先保持 package 内、目录内和相邻文件中的局部一致性；但不要继续扩散已经偏离规范的旧写法。

## 2. 格式化与版式

- `MUST`：所有 Go 源文件都符合 `gofmt` 输出。
- `SHOULD`：有 import 变化时使用 `goimports`，避免手排 import。
- `MUST`：Go 文件名默认使用全小写；多词文件名使用 snake_case，如 `count_min.go`、`querier_test.go`。
- `MUST`：不要为了“视觉对齐”手工调整空格，让 `gofmt` 决定布局。
- `MUST`：Go 没有固定行宽限制，不要为了凑 80 列或 100 列做生硬换行。
- `SHOULD`：如果某一行过长，优先缩短命名、提取变量或重构表达式，而不是机械折行。
- `SHOULD`：长字符串、URL、格式串等如果拆开会损害语义，允许保留长行。
- `SHOULD`：换行应服务于语义分组，而不是机械满足长度。

## 3. import 规范

- `MUST`：两个及以上 import 使用 `import (...)` 聚合。
- `MUST`：import 按分组组织，并使用空行分隔。
- `MUST`：标准库始终放在第一组。
- `SHOULD`：分组内按照字典序排序。
- `SHOULD`：常见分组顺序如下：
  - 标准库
  - 第三方依赖
  - 本项目或同模块的内部依赖
- `MUST`：避免不必要的 import 别名。
- `SHOULD`：仅在以下情况使用 import 别名：
  - 解决命名冲突
  - import path 末尾元素与实际包名不一致
  - 生成代码或历史包名在当前上下文中可读性较差
- `MUST`：blank import 只用于“仅需副作用”的场景。
- `SHOULD`：blank import 主要出现在 `main` 包或特定测试中。
- `AVOID`：在业务代码中使用 dot import。
- `SHOULD`：dot import 仅在测试代码因循环依赖无法放回被测包时使用。

## 4. 命名规范

### 4.1 package

- `MUST`：包名只用小写字母，不用大写、下划线和混合大小写。
- `MUST`：包名应简短、明确，并能作为路径最后一级名称。
- `SHOULD`：包名优先使用单数形式。
- `SHOULD`：包名应体现职责边界，如 `time`、`http`、`encoding/base64`。
- `AVOID`：`util`、`common`、`misc`、`api`、`types`、`interfaces`、`helper`、`model`、`testhelper` 这类无边界名称。
- `AVOID`：与标准库常用包同名，除非语义完全一致且不会一起被导入。
- `SHOULD`：不要抢占调用方常用变量名，例如 `bufio` 优于 `buf`。
- `SHOULD`：缩写要克制，只有当缩写在上下文里清晰可读时才使用，如 `fmt`、`strconv`。

### 4.2 标识符通用规则

- `MUST`：导出标识符使用 `MixedCaps`，未导出标识符使用 `mixedCaps`。
- `MUST`：不要使用 snake_case。
- `SHOULD`：命名要短，但不能牺牲可读性。
- `SHOULD`：名称离使用点越远，携带的上下文信息应越多。
- `SHOULD`：局部变量可以更短，全局变量、导出变量和跨函数共享变量应更明确。

### 4.3 缩写与首字母缩略词

- `MUST`：常见缩写保持统一大小写，如 `ID`、`URL`、`HTTP`、`JSON`、`API`、`DB`。
- `MUST`：`ServeHTTP` 而不是 `ServeHttp`，`appID` 而不是 `appId`。
- `SHOULD`：多个缩略词组合时，各缩略词内部大小写保持一致，如 `XMLAPI`、`xmlAPI`。
- `SHOULD`：像 `gRPC`、`iOS`、`DDoS` 这类在英文里自带大小写的词，按官方文档给出的习惯处理导出与未导出形式。

### 4.4 function / method

- `MUST`：函数名不要重复包名语义。
- `SHOULD`：如果 `pkg.F()` 返回 `pkg.T`，通常可以省略类型信息，如 `time.Now()`、`time.Parse()`。
- `SHOULD`：如果返回类型不是包名对应主类型，可在函数名里补充类型信息，如 `time.ParseDuration()`。
- `SHOULD`：如果某类型是包的主要入口，构造函数可以命名为 `New()`。
- `MUST`：getter 不加 `Get` 前缀，除非概念本身就是 GET / Fetch / Compute 等动作。
- `SHOULD`：如果函数会执行远程调用、复杂计算或阻塞操作，使用 `Fetch`、`Compute`、`Load` 等更明确的动词。

### 4.5 receiver

- `MUST`：receiver 名称短，通常 1 到 2 个字母。
- `MUST`：receiver 名称是类型名的缩写，并在该类型全部方法中保持一致。
- `MUST`：不要使用 `self`、`this`、`me`。
- `SHOULD`：如果 receiver 未使用，直接省略名称，不要用 `_`。

### 4.6 receiver type

- `MUST`：方法需要修改 receiver 时，使用指针 receiver。
- `MUST`：receiver 包含 `sync.Mutex` 等不可复制同步字段时，使用指针 receiver。
- `SHOULD`：大 struct / array 优先使用指针 receiver。
- `SHOULD`：若 receiver 是 map、func、chan，不要再对它们取指针。
- `SHOULD`：对 slice，如果方法不重切片也不重分配，优先不用指针 receiver。
- `SHOULD`：小型、天然值语义、无锁且无共享内部可变状态的类型可使用 value receiver。
- `AVOID`：同一类型混用 value receiver 和 pointer receiver。

### 4.7 interface

- `SHOULD`：单方法接口通常用 `-er` 命名，如 `Reader`、`Writer`。
- `MUST`：不要占用 Go 里已有强语义的方法名，除非行为完全一致，如 `Read`、`Write`、`String`、`ServeHTTP`。
- `MUST`：接口优先由消费者定义，而不是由实现方预先声明。
- `MUST`：不要仅仅为了 mock 而在实现侧导出接口。
- `SHOULD`：接口要小而精，越大越弱。
- `SHOULD`：默认遵循“accept interfaces, return concrete types”。
- `SHOULD`：只有在以下场景才优先返回接口：
  - 接口本身就是产品或协议边界
  - 需要隐藏危险实现细节以维护不变量
  - 运行期需要返回多个不同具体实现

### 4.8 variable / constant / error names

- `SHOULD`：局部变量优先短名，如 `i`、`r`、`buf`、`ctx`。
- `SHOULD`：跨作用域、跨函数、跨 goroutine 使用的变量名应更明确。
- `MUST`：常量使用 `MixedCaps`，不要用 `MAX_X`、`kMaxX`。
- `SHOULD`：常量命名表达“角色”而不是“值”，如 `MaxPacketSize` 优于 `Twelve`。
- `MUST`：可复用错误变量命名为 `errXxx` 或 `ErrXxx`。

### 4.9 file names

- `MUST`：Go 文件名默认使用全小写。
- `SHOULD`：多词文件名使用 snake_case，而不是 `MixedCaps` 或连字符。
- `SHOULD`：测试文件命名与被测文件保持可追踪关系，如 `querier.go` 对应 `querier_test.go`。

## 5. 注释与文档

### 5.1 总体要求

- `SHOULD`：Go 代码注释默认使用英文，尤其是 package comment、导出符号注释和错误文案。
- `MUST`：所有导出顶层符号都必须有 doc comment。
- `SHOULD`：非平凡的未导出类型、函数或关键内部接口也写注释。
- `SHOULD`：注释优先解释“为什么”，而不是简单复述代码“做什么”。
- `SHOULD`：复杂 API 尽量提供 runnable example，示例应放测试文件中。

### 5.2 package comment

- `MUST`：每个 package 都有包注释。
- `MUST`：同一个 package comment 只保留一份，避免多文件重复描述。
- `MUST`：包注释紧挨 `package` 声明，中间不留空行。
- `MUST`：普通包注释以 `Package <name>` 开头。
- `SHOULD`：`package main` 注释可以使用 `Command <name>`、`Binary <name>`、`Program <name>` 或类似写法。
- `SHOULD`：较长包文档放在 `doc.go`，较短可放在与包同名文件顶部。

### 5.3 doc comment

- `MUST`：导出符号注释以符号名开头。
- `MUST`：文档注释使用完整英文句子，首字母大写，句号结尾。
- `SHOULD`：即使当前符号未导出，如果它未来可能导出，也可提前按 doc comment 规范书写。
- `SHOULD`：写完文档后，用 doc preview / godoc 视角检查是否顺畅易读。

### 5.4 行内注释与代码段注释

- `MUST`：完整句子的独立注释首字母大写，并以句号结尾。
- `SHOULD`：行尾短注释可使用短语，不必强行大写和加句号。
- `SHOULD`：注释换行以可读性为准，不必机械卡列宽。
- `AVOID`：过度装饰性的 Markdown 语法；Godoc 对普通缩进更友好。

## 6. 控制流与基本写法

### 6.1 if / else

- `MUST`：如果 `if` 与 `else` 分支都以 `return` 结束，省略冗余 `else`。
- `SHOULD`：如果两个分支只给同一变量赋值，先设置默认值，再在 `if` 中覆盖。
- `SHOULD`：正常路径保持最小缩进，错误路径优先返回。

### 6.2 Indent Error Flow

- `MUST`：优先使用 guard clause 处理错误。
- `SHOULD`：遇到 `if x, err := f(); err != nil { ... } else { ... }` 且 `else` 让主路径右移时，考虑拆成两行提升可读性。

### 6.3 for range 与 blank identifier

- `MUST`：只用索引时写 `for i := range s`，不要写 `for i, _ := range s`。
- `MUST`：索引和值都不用时写 `for range s`，不要写 `for _ = range s`。
- `MUST`：不要用 `_` 忽略 error。
- `SHOULD`：blank identifier 用于明确表达“此值有意忽略”，而不是规避设计问题。

### 6.4 零值与初始化

- `SHOULD`：优先利用 Go 的零值，使类型在零值下尽量可用。
- `SHOULD`：初始化新变量时，非零值优先 `:=`，零值优先 `var`。
- `SHOULD`：当值需要稍后填充，例如 `json.Unmarshal` 目标，优先零值声明。
- `SHOULD`：空 slice 默认优先 nil slice，即 `var s []T`。
- `SHOULD`：如果 JSON 编码等场景明确需要 `[]` 而不是 `null`，可以使用非 nil 空 slice。
- `MUST`：map 在写入前必须显式初始化。
- `SHOULD`：已知容量且性能敏感时，合理使用 `make` 进行预分配；否则不要为“可能更快”提前增加复杂度。

### 6.5 in-band errors

- `MUST`：除非返回值本身就允许自然表示“无结果”，否则不要用 `-1`、`""`、`nil` 等 in-band error 表示失败。
- `SHOULD`：优先额外返回 `bool` 或 `error`。
- `SHOULD`：让错误返回值放在最后，便于调用方自然处理。

### 6.6 named result parameters 与 naked returns

- `SHOULD`：只有在返回多个同类型值、文档可读性明显提升、或 deferred closure 需要修改返回值时，才使用 named result parameters。
- `AVOID`：仅仅为了少写一行局部变量而命名返回值。
- `MUST`：不要只是为了启用 naked return 而命名返回值。
- `SHOULD`：naked return 只用于非常短、非常直接的函数。
- `AVOID`：在中长函数里使用 naked return，这会降低可读性并增加维护成本。

### 6.7 pass values

- `SHOULD`：默认按值传递参数；只有在需要共享修改、避免高成本复制、或表达“可为空”语义时才使用指针。
- `AVOID`：仅为了“省几个字节拷贝”就把小对象或基础类型全部改成指针。
- `MUST`：不要传递指向 interface 的指针。
- `SHOULD`：像 `string`、`time.Time`、小 struct 这类天然值语义对象，优先直接传值。
- `SHOULD`：如果函数需要修改输入、或输入持有锁/大数组/大 struct，优先传指针。

### 6.8 map iteration

- `MUST`：涉及 SQL 构造、序列化输出、签名计算或测试断言时，不要依赖 map 的遍历顺序。
- `SHOULD`：如果输出顺序会影响日志、查询文本、错误复现或测试稳定性，先取 key 列表再排序。

## 7. 错误处理

### 7.1 基本原则

- `MUST`：任何返回 error 的调用都必须被处理、返回，或在极少数情况下转成 panic。
- `MUST`：常规业务流程中不要使用 panic。
- `SHOULD`：error 是值，错误处理逻辑同样要追求可读性和复用性。

### 7.2 错误字符串

- `MUST`：错误字符串不以大写字母开头，除非首词是专有名词或缩略词。
- `MUST`：错误字符串不以标点结尾。
- `MUST`：错误字符串使用 ASCII 英文，不使用中文或拼音。
- `SHOULD`：如果缺少上下文，可以统一加 package 前缀，如 `gif: invalid pixel value`。

### 7.3 错误变量与自定义错误

- `SHOULD`：只出现一次且不需要判定的错误，直接用 `errors.New` 或 `fmt.Errorf` 返回。
- `SHOULD`：需要多处复用或被外部捕获的错误，定义为具名错误变量。
- `SHOULD`：一个文件只有一个错误值时，可放在文件顶部；多个错误值按逻辑块就近放置。
- `SHOULD`：复杂错误类型放在 `error.go`。
- `MUST`：自定义错误类型以 `Error` 结尾，并实现：
  - `Error() string`
  - `Unwrap() error`
  - 持有底层 `Err error` 字段

## 8. 生命周期与并发边界

- `SHOULD`：`Start`/`Stop` 一类生命周期 API 优先由实现方自行启动 goroutine，并让调用方通过返回值和 `Stop` 感知失败与退出。
- `MUST`：库代码不要用 `log.Fatal`/`log.Fatalf` 处理常规错误；把错误返回给调用方，由 `main` 包或顶层 runner 决定如何退出。
- `SHOULD`：停止流程应尽量幂等；如果组件支持 `Stop`，重复调用不应引发 panic。

### 7.4 wrap / unwrap / 判定

- `MUST`：需要保留原始错误链时，使用 `fmt.Errorf("...: %w", err)`。
- `MUST`：判定错误类型或错误值时，用 `errors.Is` / `errors.As`。
- `SHOULD`：只在确实需要向下兼容旧实现时直接调用 `errors.Unwrap`。
- `AVOID`：通过比较错误字符串进行错误分类。

### 7.5 API 文档中的错误约定

- `SHOULD`：文档化重要的 sentinel error 和自定义错误类型。
- `SHOULD`：如果函数在 `ctx` 取消时返回的不是 `ctx.Err()`，应在注释里说清楚。
- `SHOULD`：如果某个资源需要调用方关闭，应在 doc comment 明确写出 cleanup 责任。

## 8. panic / recover / iota

### 8.1 panic

- `AVOID`：业务代码中使用 panic。
- `SHOULD`：启动阶段遇到不可恢复错误时，可在 `init` 或 `main` 中 panic。
- `SHOULD`：若错误可被调用方处理，返回 error 而不是 panic。

### 8.2 recover

- `MUST`：`recover` 只能在 `defer` 的函数里生效。
- `MUST`：`recover` 只对当前 goroutine 生效。
- `SHOULD`：如果恢复 panic 后继续运行，至少记录足够上下文；必要时记录调用栈。
- `SHOULD`：如果不是可识别、可屏蔽的 panic，恢复后重新 panic。

### 8.3 iota

- `SHOULD`：`iota` 主要用于内部枚举常量，不建议在业务协议或外部 API 里滥用。
- `MUST`：枚举值使用显式自定义类型。
- `SHOULD`：相关枚举在同一个 `const` 组中声明。

## 9. context、并发与生命周期

### 9.1 context

- `MUST`：使用 `context.Context` 时，参数名统一为 `ctx`，并放在第一个参数位置。
- `SHOULD`：即使暂时觉得“用不到”，也优先把 `ctx` 沿调用链透传下去。
- `MUST`：不要把 `Context` 作为 struct 字段存储，除非方法签名必须满足第三方接口。
- `MUST`：不要自定义 `Context` 类型，也不要把接口类型替代为其他抽象。
- `SHOULD`：业务数据优先通过普通参数、receiver 或结构体字段传递，不滥用 `ctx.Value`。
- `SHOULD`：如果 API 对 context 的 deadline、value、生命周期有非直觉要求，必须文档化。

### 9.2 goroutine lifetimes

- `MUST`：启动 goroutine 时，要让其退出条件清晰可推断。
- `MUST`：避免因 channel 阻塞导致 goroutine 泄漏。
- `SHOULD`：并发代码尽量简单，使 goroutine 生命周期一眼可见。
- `SHOULD`：如果退出逻辑不显而易见，必须在代码或注释中说明何时退出、为何退出。

### 9.3 synchronous functions

- `SHOULD`：默认优先设计同步函数。
- `SHOULD`：如无必要，不要把并发复杂度强加给调用方。
- `SHOULD`：如果调用方需要并发，可由调用方在外层起 goroutine。

### 9.4 copying

- `MUST`：不要随意复制带内部引用语义或指针方法集的外部 struct。
- `SHOULD`：如果某类型的方法主要定义在 `*T` 上，默认不要复制 `T` 值。

### 9.5 randomness

- `MUST`：密钥、token、nonce、一次性验证码等安全敏感随机数使用 `crypto/rand`。
- `AVOID`：用 `math/rand` 生成安全敏感数据。

## 10. 接口与 API 设计

- `MUST`：不要在实现侧预先定义“也许以后会用到”的接口。
- `MUST`：不要为了测试而导出接口或测试替身类型。
- `SHOULD`：先写具体类型；只有在出现真实替换需求后再提炼接口。
- `SHOULD`：消费者定义只依赖自己真正使用的方法集。
- `SHOULD`：当接口本身是协议边界时，接口文档要像“用户手册”一样完整，说明行为、边界、并发与错误语义。
- `SHOULD`：如果函数返回资源句柄、后台 worker、stream、ticker 或连接，必须写清楚如何关闭、何时关闭、取消后语义如何。
- `AVOID`：仅为了“看起来更抽象”而包装已有生成代码接口或 RPC 客户端。

## 11. 测试规范

### 11.1 基本风格

- `SHOULD`：测试代码也按生产代码标准维护。
- `SHOULD`：表驱动测试在能减少重复并突出关键信息时优先使用。
- `SHOULD`：大型测试表或相邻字段类型相同的 struct literal 优先写字段名。
- `SHOULD`：优先让 `Test` 函数自己表达断言，而不是堆大量 assertion helper。

### 11.2 失败信息

- `MUST`：失败信息要包含输入、实际值和期望值。
- `SHOULD`：保持消息顺序为“got / want”。
- `SHOULD`：复杂对象比较优先输出可读 diff，例如 `cmp.Diff` 风格。

### 11.3 t.Error / t.Fatal

- `SHOULD`：默认优先 `t.Error`，让一个测试尽量报告更多独立问题。
- `SHOULD`：只有在后续步骤无法继续时，才用 `t.Fatal` / `t.Fatalf`。
- `SHOULD`：表驱动测试中，如果某个 case 失败但不影响其他 case，优先 `t.Error` 加 `continue`，或使用 `t.Run` 隔离为子测试。
- `MUST`：不要在其他 goroutine 中调用 `t.Fatal` / `t.FailNow`。

### 11.4 helpers 与 cleanup

- `SHOULD`：测试 helper 若会报告失败，使用 `t.Helper()`。
- `SHOULD`：setup / cleanup helper 发生失败时，允许在 helper 内直接 `t.Fatal`，但错误信息要给出上下文。
- `SHOULD`：能用 `t.Cleanup` 简化资源回收时优先使用。

### 11.5 集成测试

- `SHOULD`：组件间通过 HTTP / RPC 交互时，优先使用真实 transport 连接测试版本后端，而不是手写假客户端。
- `SHOULD`：尽量通过公开 API 测试真实实现，不要为了 mock 而扭曲生产 API 设计。

## 12. Go Modules 与版本管理

### 12.1 基础要求

- `MUST`：每个模块根目录只有一个 `go.mod`。
- `MUST`：`go.mod` 与 `go.sum` 提交到版本控制。
- `SHOULD`：依赖变更后运行 `go mod tidy`。
- `SHOULD`：必要时用 `go mod why -m`、`go mod graph` 分析依赖来源。

### 12.2 版本与发布

- `MUST`：发布版本 tag 使用 `vX.Y.Z`。
- `SHOULD`：`v0.x.x` 视为实验阶段，不兼容改动可以不升 major，但建议升 minor。
- `MUST`：`v1.x.x` 起遵守语义化版本。
- `MUST`：`v2+` 模块使用 semantic import versioning，module path 追加 `/vN`。
- `MUST`：从 `v1` 升到 `v2+` 时，在仓库中新建 `vN/` 目录，复制代码与 `go.mod`，并把 `module path` 改成带 `/vN` 的路径。

### 12.3 依赖方 import 规则

- `MUST`：依赖 `v0` / `v1` 模块时，import path 不带版本后缀。
- `MUST`：依赖 `v2+` 模块时，import path 带 `/vN`。
- `SHOULD`：升级依赖版本后，及时修复 API 不兼容问题并补充测试。

## 13. 适用于 Codex 的执行清单

每次让 Codex 修改 Go 代码前后，都按以下顺序检查：

1. 先看清 package 边界、局部命名和已有风格，优先保持一致。
2. 先选最简单的实现，不预先引入接口、抽象层或并发。
3. 写代码时同步考虑：
   - 命名是否符合 Go 习惯
   - 导出符号是否需要英文注释
   - 错误是否被正确处理与 wrap
   - context 是否首参透传
   - goroutine 是否有退出条件
4. 改完后检查：
   - `gofmt` / `goimports`
   - import 分组
   - 是否引入了冗余 `else`
   - 是否错误地忽略了 error
   - 是否误用了 panic、dot import、blank import、提前接口化
5. 能验证时执行：
   - 改动包的 `go test`
   - 影响面较大时 `go test ./...`

## 14. 参考来源

- [Google Go Style Guide - Guide](https://google.github.io/styleguide/go/guide)
- [Google Go Style Guide - Decisions](https://google.github.io/styleguide/go/decisions)
- [Google Go Style Guide - Best Practices](https://google.github.io/styleguide/go/best-practices)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Blog - Package names](https://go.dev/blog/package-names)
- [Go Blog - Using Go Modules](https://go.dev/blog/using-go-modules)
- [Go Blog - Migrating to Go Modules](https://go.dev/blog/migrating-to-go-modules)
- [Go Blog - Publishing Go Modules](https://go.dev/blog/publishing-go-modules)
- [Go Blog - Go Modules: v2 and Beyond](https://go.dev/blog/v2-go-modules)
