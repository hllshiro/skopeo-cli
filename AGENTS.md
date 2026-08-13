# skopeo-cli — Agent Instructions

## 发版规范（Release）

每次发版**必须**撰写 release notes，且 release notes **必须使用中英双语**。

### 发版流程

1. 更新 `main.go` 中的 `version` 常量
2. 提交并推送代码
3. 更新 `.github/RELEASE_NOTES.md`（双语，内容见下）
4. 打标签并推送触发 CI 自动构建与发布：

   ```bash
   git tag -a vX.Y.Z -m "vX.Y.Z: <summary>"
   git push origin vX.Y.Z
   ```

   发布由 GitHub Actions 完成（`.github/workflows/build.yml`），
   release notes 通过 `body_path` 自动引用 `.github/RELEASE_NOTES.md`。

### Release Notes 要求

- **必须写**：每次发版都要写 release notes，不可省略。
- **必须双语**：中英双语，两种语言都要完整，不得只写一种。
- **内容结构**（每个版本至少覆盖以下小节）：
  - 版本标题（如 `## v2.0.0 — ...`）
  - Highlights（亮点）
  - Changes（变更，含 CLI 变更）
  - Fixes（修复）
  - Breaking changes（破坏性变更，如有）
  - Upgrading（升级指引）
- **存放位置**：`.github/RELEASE_NOTES.md`

### 示例

```markdown
## v2.0.0 — Complete rewrite in Go / 完全用 Go 重写

### Highlights / 亮点

- ...

### Changes / 变更

- ...

### Fixes / 修复

- ...

### Breaking changes / 破坏性变更

- ...

### Upgrading / 升级指引

...
```
