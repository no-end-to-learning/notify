# Agent Guide

本文件给 AI 代理使用，说明本仓库内必须遵守的开发和提交规则。面向人的项目说明见 [README.md](README.md)。

## 基本规则

- 默认在 `master` 上开发；提交前确认 `git status --short` 只包含本次任务相关文件。
- 不要回滚用户已有改动；遇到无关脏工作区直接忽略，除非它直接阻塞当前任务。

## 提交规则

- 提交信息统一使用中文，subject 和 body 都必须写中文。
- 格式：`<type>(<scope>): <中文摘要>`；type 用 `feat`/`fix`/`docs`/`refactor`/`chore`/`build`/`ci`，不随意新增。
- 需要 body 时主题和正文之间保留一个空行，正文用中文条目说明关键改动。
- 需要 body 时用单个 `git commit -m $'...\n\n...'` 传入完整信息，避免交互式编辑器或多次 `-m` 产生多余换行。
- 不要加入 AI 署名、生成标识、协作者 trailer 或工具标记。

```bash
git commit -m $'feat(queue): 添加队列重试配置\n\n- 支持通过环境变量配置重试次数。\n- 更新队列初始化逻辑。'
```

## 常用校验

```bash
go test ./...
go build -o notify .
```

CI 会运行测试并构建多架构镜像；改动后仍需手动起服务验证接口行为。

## 禁止事项

- 不要提交 `.env`、真实的 `APP_FEISHU_SECRET`/`APP_TELEGRAM_BOT_TOKEN`。
- 不要绕过限频器直接发送消息，改动限频/重试逻辑时要保留"每个 channel+target 独立队列"的语义。
