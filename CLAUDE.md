# Git 提交流程

## 提交信息格式

- **type类型**：feat、fix、docs、style、refactor、perf、test、build、ci、chore、revert
- **描述标题**：简洁明确，不超过30字符
- **描述内容**：分点列出，每个变更点用一句话概括

## 禁止事项

- 严禁包含AI相关标识（如Claude、AI、ChatGPT等）
- 严禁描述对话过程（如"根据用户要求"、"用户想要"等）
- 严禁添加Co-Authored-By

## 提交流程

1. 查看变更：`git status` 和 `git diff`
2. 查看提交风格：`git log --oneline -5`
3. 分析变更内容，生成commit信息
4. **用户确认commit信息后**再执行：
   - 添加文件：`git add -A`
   - 检查：`git status`
   - 提交：`git commit -m "type: 描述"`
   - 推送：`git push`

## 示例

```
git commit -m "feat: Add user login

- Enable username and password login
- Add JWT token verification"
```
