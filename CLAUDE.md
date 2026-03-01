# Git 提交流程

## 提交信息格式

- **type类型**：feat、fix、docs、style、refactor、perf、test、build、ci、chore、revert（**仅使用上述type**）
- **描述标题**：简洁明确，不超过30字符（**Commit标题必须使用英文**）
- **描述内容**：分点列出，每个变更点用一句话概括（**Commit描述必须使用英文**）

## 严厉禁止事项，最高级别禁止

- 严禁包含AI相关标识（如Claude、AI、ChatGPT等）
- 严禁描述对话过程（如"根据用户要求"、"用户想要"等）
- 严禁在Commit信息当中添加Co-Authored-By

## 提交流程

1. 查看当前变更：`git status` 和 `git diff`
2. 查看历史提交风格：`git log --oneline -5`
3. 分析变更内容，生成commit信息（**Commit信息格式可参考示例**）
4. **使用 AskUserQuestion 工具与用户以交互形式确认commit信息**后再执行提交：
   - 将commit信息包含在AskUserQuestion消息中，选项为"确定"
   - 用户选择"确定"后，继续使用git工具执行以下命令：
     - **务必使用** `git add -A` 添加所有文件
     - 再次检查：`git status`
     - 提交Commit：`git commit -m "type: title -description1 -description2"`
     - 推送到远程：`git push`

## Commit信息示例

```
git commit -m "feat: Add user login

- Enable username and password login
- Add JWT token verification"
```
