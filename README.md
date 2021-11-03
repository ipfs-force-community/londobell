# LondoBell

## 开发注意事项
- 参与者必须通过执行 `make dev-env` 向本地的开发环境中注入必要的内容，如 git-hooks 等

- 参与者必须确保自己的开发工具包含以下代码格式化和检查的集成：
  - 自动的 gofmt、goimports 格式化
  - 自动添加文件末尾的新行
  - [golangci-lint 的集成](https://golangci-lint.run/usage/integrations/)

- 代码的提交必须经由 Pull Request， 并应当始终确保：
  - 可编译
  - 可通过 go test 检查
　- 可通过 golangci-lint 检查

- 原则上，Pull Request 要求聚焦，多个不相关的改动应当分到不同的 PR 中去; 不相关的改动包含但不限于：
  - 同时对多个不相依赖的模块或逻辑进行修改
  - 业务需求和重构需求
  - 逻辑性的代码变更和代码规整

- Pull Request 的 Review 需要关注：
  - 自动检查项的通过情况
  - 代码逻辑
  - 测试用例的编写和补充
