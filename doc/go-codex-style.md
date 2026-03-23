# Go 代码规范 规则清单

## 1. Formatting

- [MUST] 统一依赖 `gofmt` 和 `goimports`；`import` 至少要区分“标准库”和“其他包”两组，也可以细分为依赖包、项目包、Outer/Inner 包等 3 组或 4 组，但同一项目必须长期保持一种一致的分组方式。
- [SHOULD] 只有在换行能提升可读性时才换行，不要只因为行长看起来过长就机械断行；原文没有硬性行长上限。

## 2. Commentary

- [SHOULD] 大多数情况下使用行注释 `//`，包括 package comment；只有在表达式内部这类局部场景才使用块注释 `/* */`。
- [SHOULD] 注释优先使用英文，以保证跨团队协作和代码审查的一致可读性。

### 2.1 Package Comment

- [RECOMMENDED] 在包顶部写简短包注释，直接说明包的职责、边界和用途，而不是重复包名。
- [OPTIONAL] 当包的初始化方式、调用方式或用途不够直观时，可以在包注释里附一个最小示例。

### 2.2 Function Comment

- [MUST] 导出函数必须写用法注释，除非函数名本身已经完全自解释；注释应至少帮助调用方理解前置条件、输入、输出、性能影响和错误处理方式。
- [RECOMMENDED] 函数注释以函数名开头，便于文档工具生成和读者快速定位。
- [MAY NOT] 只有当函数名和行为都非常直白、没有额外前提或坑点时，自解释的函数可以不写注释。

### 2.3 Function Argument Comment

- [RECOMMENDED] 当函数参数含义不直观时，可用 `/*argument=*/` 添加参数注释；但更优先通过具名常量、配置结构体或中间变量让调用点自己变得清楚。

### 2.4 Implementation Comment

- [SHOULD] 实现注释只解释棘手、不明显、重要或有陷阱的部分，帮助维护者理解“为什么这样写”。
- [SHALL NOT] 不要写“把代码翻译成自然语言”的显然注释，避免制造噪音。

### 2.5 Variable Comment

- [SHOULD] 如果变量名在当前上下文中不够自解释，就为变量声明补充注释，说明它的业务语义、约束或特殊含义。

### 2.6 TODO Comment

- [SHOULD] 每个 TODO 都写明负责人，否则 TODO 很容易长期悬空并被遗忘。
- [RECOMMENDED] TODO 更推荐指向 JIRA ticket，而不是直接指向个人，因为 ticket 更易转派、追踪和审计。

## 3. Names

- [MUST] 大多数命名使用 `CamelCase/camelCase`，并尽量避免下划线；项目内命名风格必须统一。
- [SHOULD] 命名尽量短且准确，既不要冗长堆砌上下文，也不要为了短而牺牲含义。
- [SHOULD] 缩写要克制，只在读者普遍熟悉且不会引起歧义时使用，避免自造缩写或生僻词。

### 3.1 Package

- [MUST] 同一项目内的包名必须唯一；如果两个包都想用同名，通常说明命名过泛或职责边界重叠，需要重审设计。
- [MUST] 包名必须全小写，不能包含大写字母或下划线。
- [MUST] 包名必须用单数形式，例如用 `httputil`，不要用 `httputils`。
- [SHOULD] 包名应简短但能代表职责，命名时描述“这个包提供什么”，而不是“这个包包含什么”；尽量避免顶层包名用 `common`、`util` 这种过宽名称。

### 3.2 Function

- [MUST] 函数名必须使用 `MixedCaps` 或 `mixedCaps`；导出函数用 `MixedCaps`，非导出函数用 `mixedCaps`，测试函数为了分组可以例外地包含下划线。
- [SHOULD NOT] 函数名不要重复表达包名或包内容，因为调用点本来就会带上包名前缀。
- [RECOMMENDED] 函数名优先用动词或动词短语，让读者一眼知道它在做什么动作。

### 3.3 Variable

- [SHOULD] 变量名应短于长，只保留真正区分语义的部分，避免无意义的冗余词。
- [SHOULD NOT] 不要把类型写进变量名，例如避免 `nicknameStr`、`userSlice`；只有在需要刻意区分转换前后类型时才值得这么做。
- [MUST] 首字母缩略词必须全大写，例如 `XMLRequest`；如果缩略词位于未导出标识符开头，则保持小写风格，例如 `xmlRequest`。
- [SHOULD] 如果使用短变量名，如 `i`、`j`、`v`，它们在相似上下文里的语义应保持一致，不要同一类循环里随意换成另一组短名。

### 3.4 Receiver

- [SHOULD] receiver 名使用类型名的一到两个字母缩写，并在同一类型的所有方法中保持一致。
- [SHOULD NOT] receiver 名不要用 `self`、`this`、`me` 这类泛化名字，因为它们既不地道，也不表达具体类型身份。

## 4. Control Structures

### 4.1 If

- [SHOULD] 能不用 `else` 就尽量不用 `else`，优先通过提前返回或先处理异常分支来降低嵌套深度。
- [RECOMMENDED] 相比 `if/else`，更推荐先初始化默认值，再在 `if` 分支中按需改写，这样主流程更线性、更易读。

### 4.2 For

- [SHOULD] `for` 循环里只声明你真正需要的变量；只用 value 就写 `_, value := range ...`，只用 key 就直接写 `for key := range ...`。

### 4.3 Switch

- [RECOMMENDED] 多分支判断优先用 `switch` 替代一串 `if`，在无限循环里处理多种状态时也优先用 `switch` 或对应的分支结构组织逻辑。
- [SHOULD] 谨慎使用 `fallthrough`；它会跳过下一个 case 的条件判断，容易制造隐蔽行为，因此一旦使用就必须特别注意分支顺序，而且不能出现在最后一个 case 中。

## 5. Functions

### 5.1 Length

- [SHOULD] 函数尽量小而聚焦；当一个函数开始变得难改、难测、难复用时，应主动拆成更小的辅助函数。

### 5.2 Grouping and Ordering

- [SHOULD] 同一文件中的函数按大致调用顺序排列，让读者可以顺着执行路径自上而下理解代码。
- [SHOULD] 同一文件中的函数按 receiver 分组，把同一类型的方法放在一起。
- [SHOULD] 导出函数放在文件靠前位置，并位于 `struct`、`const`、`var` 定义之后，优先暴露对外 API。
- [MAY] `newXYZ()` / `NewXYZ()` 可以放在类型定义之后、该 receiver 的其他方法之前，作为理解类型的入口。
- [SHOULD] 普通工具函数放在文件后部，不要打散按 receiver 建立的主结构。

### 5.3 Context

- [SHOULD] 只要函数使用 `context`，就把它作为第一个参数，以保持整个调用链一致。
- [SHOULD NOT] 不要把 `context.Context` 存成 struct 成员，而应把 `ctx` 显式作为参数传入；只有当方法签名必须兼容外部接口时才可例外。

### 5.4 Named Result Parameters

- [RECOMMENDED] 当具名返回值能明显提升可读性时使用它，尤其是在返回多个同类型值、需要直接表达每个返回值含义时。
- [RECOMMENDED] 只有在函数足够短且逻辑足够简单时才使用裸返回，避免它反过来损害理解成本。

### 5.5 Defer

- [SHOULD] 文件、锁等资源的清理优先使用 `defer`，这样既能避免遗漏释放，也能让“获取资源”和“释放资源”的代码保持邻近。

### 5.6 Pass Values vs Pointers

- [RECOMMENDED] 大结构体或未来可能增长的结构体优先传指针，其余场景优先传值；不要为了省几个字节滥用指针，尤其避免 `*string`、`*interface` 这类通常多余的写法，而对 slice、map、channel、string、func、interface 也要意识到它们本身已带引用语义。

## 6. Data

### 6.1 nil is a valid slice

- [RECOMMENDED] 零值 slice（`var nums []int`）应视为可立即使用，不必先 `make()`，因为 `nil` slice 本身就是合法的空 slice。
- [RECOMMENDED] 需要返回“空结果”时，优先返回 `nil`，不要显式返回长度为 0 的 slice。

## 7. Initialization

### 7.1 Constants

- [MUST] 常量的定义表达式必须是编译期可求值的常量表达式；Go 常量只能是数字、字符、字符串或布尔值，不能依赖运行期计算。

### 7.2 Initializing Structs

- [SHOULD] 初始化结构体时几乎总是显式写出字段名，以提高可读性并降低字段顺序变化带来的风险。
- [SHOULD] 如果声明结构体时省略了所有字段，就用 `var user User`，不要写成 `user := User{}`，以明确表达“这里只是在拿零值”。

### 7.3 Initializing Struct References

- [SHOULD] 初始化结构体指针时使用 `&T{}`，不要使用 `new(T)`，这样风格更一致，也更便于在创建时直接填字段。

### 7.4 Initializing Maps

- [SHOULD] 空 map 或通过代码逐步填充的 map 使用 `make(..)` 初始化，以清楚区分“nil map 声明”和“可写 map 初始化”，并为以后添加 size hint 留出空间。

### 7.5 The init function

- [SHOULD] 能不用 `init()` 就不要用 `init()`；如果必须用，也要避免依赖其他 `init()` 的执行顺序或副作用，避免访问全局/环境状态，并避免 I/O。

## 8. Methods

### 8.1 Receiver Names

- [SHOULD] receiver 名优先使用类型名的一到两个字母缩写，让它既短又能看出对应类型。

### 8.2 Receiver Type

- [RECOMMENDED] receiver 类型拿不准时优先用指针；凡是需要修改 receiver、包含锁字段、对象较大、需要共享外部变化，或希望避免值复制语义误导时，都应使用指针 receiver，且不要在同一类型上混用值 receiver 和指针 receiver。

## 9. Interfaces and Other Types

- [RECOMMENDED] 对会在系统中广泛传递的能力定义接口，`setup` / `new..` 这类函数优先返回接口而不是具体结构体；真正实现接口的仍然是结构体，这样更利于替换、测试和编译期校验。

## 10. The Blank Identifier

### 10.1 Blank Imported Package

- [MUST] 导入包时必须使用显式包名或别名，让每个依赖都清楚地作为命名空间出现。
- [MUST NOT] 不要用空白标识符 `_` 导入包，因为它只会触发初始化副作用；只有少数必须靠空导入注册驱动的基础库场景可以例外。

### 10.2 Ignore Values

- [MUST NOT] 不能用空白标识符忽略函数返回的 error，错误必须被显式处理，否则会直接损害稳定性、可靠性和可测试性。

### 10.3 Interface Check

- [SHOULD] 使用空白标识符做接口实现检查，例如 `var _ Iface = (*impl)(nil)`，把“必须实现某接口”的约束提前到编译期暴露。

## 11. Errors

### 11.1 Error Wrapping

- [RECOMMENDED NOT] 给错误补充上下文时不要滥用 `failed to` 这类显然且会层层堆叠的表述；只补最必要的上下文，例如 `new store: %w`。

### 11.2 Don't Panic

- [MUST] 生产环境代码必须避免 `panic`；除了启动或初始化阶段遇到绝对无法恢复的错误外，正常错误都应通过 `error` 返回给调用方决定如何处理。

### 11.3 Recover

- [SHOULD] 对可能触发 panic 的代码，包括数组越界、类型断言失败等运行时错误，使用 `recover` 做局部保护；注意它只在 `defer` 中生效，且只影响当前 goroutine。

## 12. Concurrency

### 12.1 Goroutine

- [MUST] 每个 goroutine 都必须有明确的退出时机和退出条件，防止 goroutine 泄漏；常见手段是传 `context` 或显式等待退出。
- [SHOULD] 执行复杂任务的 goroutine 应具备恢复策略，例如在内部 `defer recover()` 并记录日志，以避免 panic 扩散成进程级问题。

### 12.2 Use Channels

- [SHOULD] 优先通过 channel 通信，而不是直接共享内存，让并发代码的协作关系更清楚、更可扩展。
- [SHOULD] channel 默认先使用无缓冲版本，只有在有明确指标或 profiling 证据表明这里是瓶颈时，再改为有缓冲或其他更激进的优化方式。

### 12.3 Avoid Explosion

- [MUST NOT] 不要无限制地创建 goroutine，尤其是在高流量在线接口的关键路径上；需要通过 goroutine pool、限流或类似机制控制并发规模。

### 12.4 Initialization

- [SHOULD] 一次性初始化或单例模式优先使用 `sync.Once`，不要依赖 `init()`，尤其当初始化逻辑可能失败时更应如此。

### 12.5 Standard Library

- [SHOULD] `sync.Mutex` / `sync.RWMutex` / `sync.Cond` 等同步原语只在构建基础库且对性能敏感的场景中作为主要方案；普通业务并发优先考虑 channel 风格。
- [RECOMMENDED] 关键路径上频繁创建和销毁的复杂对象优先考虑 `sync.Pool`，以降低分配和 GC 压力。
- [RECOMMENDED] 在合适场景下充分利用 `sync.Map` 和 `atomic.Value`，避免手写不必要的并发控制样板代码。
