# AGENTS.md

本仓库中的 Codex 在修改 Go 代码时，除系统级通用约束外，还必须遵守以下仓库级规则。

## 语言与适用范围

- 与用户沟通使用中文。
- Go 代码中的包注释、导出符号注释、错误字符串默认使用英文。
- 优先保持现有架构、API、行为和目录边界，不做无关重构。
- 优先遵循同文件、同 package 的局部一致性；但不要新增、扩散或固化违反本文件硬规则的写法。

## 核心原则

- 先保证清晰，再考虑简洁、性能和抽象。
- 优先使用 Go 社区的标准做法，不为“设计感”引入额外层次。
- 复杂性只有在确有收益时才允许出现，并且必须配套注释、文档或测试。

## Go 代码硬规则

1. 格式化
   - 所有改动过的 Go 文件必须运行 `gofmt`。
   - import 发生变化时优先运行 `goimports`。
2. import
   - import 使用分组并按字典序排序。
   - 标准库始终放第一组。
   - import 别名仅用于解决冲突或 import path 末尾名称与包名不一致。
   - 空白导入仅用于 `main` 包或确有需要的测试。
   - 点导入仅允许用于受循环依赖限制的测试。
3. 命名
   - 包名使用小写、简短、单数形式，可作为路径 base name。
   - 避免 `util`、`common`、`misc`、`api`、`types`、`interfaces`、`helper`、`model` 之类泛化命名。
   - 标识符使用 `MixedCaps` 或 `mixedCaps`，不要使用 snake_case。
   - 常见缩写保持统一大小写，如 `ID`、`URL`、`HTTP`、`JSON`、`API`、`RPC`、`DB`。
   - getter 不使用 `Get` 前缀，除非语义本身就是 HTTP GET 或远程抓取。
   - receiver 使用 1 到 2 个与类型相关的短缩写，并在该类型全部方法中保持一致；不要使用 `self`、`this`。
4. 注释与文档
   - 每个 package 都要有包注释。
   - 普通包注释以 `Package <name>` 开头；命令包注释以 `Command`、`Binary` 或 `Program <name>` 开头。
   - 所有导出符号都必须有英文 doc comment，且以符号名开头。
   - 非平凡的未导出类型或函数也应补充注释。
   - 注释优先解释“为什么”，避免复述代码“做了什么”。
   - 独立成句的注释首字母大写并以句号结尾；行尾短注释可以是短语。
5. 控制流
   - 优先 guard clause，优先缩进错误路径，保持正常路径最小缩进。
   - 两个分支都 `return`，或都只给同一变量赋值时，不写冗余 `else`。
   - `for range` 中未使用的索引或值不要保留多余 `_`。
   - 没有语义差异时，空 slice 优先 `var s []T` 而不是 `[]T{}`。
   - 不要引入 in-band error；需要表示“无结果”或“失败”时，优先额外返回 `bool` 或 `error`。
6. 错误处理
   - 不要忽略 error，禁止用 `_` 丢弃 error。
   - 业务代码不要用 panic 处理常规错误。
   - 错误字符串默认小写开头、不以标点结尾、不使用中文。
   - 需要透传上游错误时使用 `fmt.Errorf("...: %w", err)`。
   - 判断错误使用 `errors.Is` / `errors.As`，不要依赖脆弱的字符串比较。
   - 可复用错误值统一命名为 `errXxx` 或 `ErrXxx`。
   - 复杂自定义错误优先放到 `error.go`，并实现 `Error()` 与 `Unwrap()`。
7. API 与抽象
   - 接口优先由消费者定义，保持小而精，不要为了 mock 预先造接口。
   - 默认遵循“accept interfaces, return concrete types”。
   - 使用 `context.Context` 时，参数名统一为 `ctx`，并放在第一个参数位置。
   - 不要把 `Context` 存进 struct，除非签名受外部接口约束。
   - 优先设计同步函数；如果必须异步，必须让 goroutine 生命周期、退出条件和清理责任清晰可见。
   - 明确文档化调用方的清理责任、取消语义和重要错误类型。
8. 数据与 receiver
   - 不要仅为了“少拷贝几个字节”而把参数一律改成指针。
   - 需要修改 receiver、避免复制含锁结构体、结构体较大、或要保持共享可见性时，使用指针 receiver。
   - 小型、天然不可变、无锁且无共享内部指针的值类型可使用 value receiver。
   - 同一类型不要混用 value receiver 和 pointer receiver，除非有极强理由。
9. 并发与测试
   - 新起 goroutine 时，必须让退出条件明确，避免泄漏。
   - 安全敏感随机数必须使用 `crypto/rand`，不要用 `math/rand` 生成密钥、token 或 nonce。
   - 测试失败信息要包含输入、实际值和期望值。
   - 表驱动测试在能提升可读性时优先使用。
   - 不要在其他 goroutine 中调用 `t.Fatal` / `t.FailNow`。
10. Go modules
   - 依赖变更后同步更新 `go.mod` / `go.sum`，并优先运行 `go mod tidy`。
   - 发布 tag 使用 `vX.Y.Z`。
   - `v2+` 模块路径必须带 `/vN` 后缀。

## 完成前检查

- 对改动过的 Go 文件执行格式化。
- 检查 import 分组、命名、注释、错误处理和 context 用法是否符合规范。
- 能运行测试时，至少执行与改动直接相关的 `go test`；影响面较大时优先执行 `go test ./...`。

## 详细规范

完整规范、例外说明、示例与来源链接见 `doc/go-codex-style.md`。
