# 智颖API监控企业版 UI 裁剪上线复核

日期：2026-09-03  
范围：前端 UI 展示裁剪，不改后端、数据库、计费、渠道、Key、日志和模型调用链路。

## 本次改动目标

- 产品展示名固定为 `智颖API监控`。
- 未登录访问根路径 `/` 进入登录页。
- 已登录访问根路径 `/` 进入 `/dashboard/models`。
- 登录页改为简洁账号密码登录页。
- 登录页不展示注册、忘记密码、Passkey、OAuth、微信登录入口。
- 顶部栏不展示主页、控制台、文档、关于、搜索、公告通知、语言切换、主题设置。
- 禁用已登录页面的命令面板快捷入口，避免通过 `Ctrl/⌘+K` 暴露隐藏菜单。
- 左侧栏只保留：
  - 数据看板
  - API 密钥
  - 使用日志
  - 任务日志
  - 钱包
  - 个人资料
  - 渠道
  - 用户
  - 系统信息
- 数据看板只保留：
  - 模型调用分析
  - 用户统计
- 系统设置入口不在常规界面展示。

## 重要边界

- 隐藏入口不等于删除功能。
- `/system-settings/*`、模型元信息、兑换码、订阅、任务插件等路由文件仍保留。
- 直接访问受保护页面时，仍由原有角色权限控制。
- 不改变线上员工使用的统一 Base URL、虚拟 Key、计费规则和消耗统计逻辑。
- 本次 UI 裁剪面向 P0 快速上线：优先保证员工常规使用路径干净，底层管理能力先保留，后续可按需要继续收窄直达路由权限。

## 本次涉及的主要代码点

- `web/index.html`：页面标题改为 `智颖API监控`，移除默认 favicon 引用。
- `web/src/lib/constants.ts`、`web/src/main.tsx`：前端启动时固定企业品牌名，避免旧系统配置覆盖页面标题。
- `web/src/routes/index.tsx`、`web/src/routes/(auth)/sign-in.tsx`、`web/src/features/auth/**`：根路径和登录后默认进入模型调用分析；登录页裁剪为账号密码表单。
- `web/src/routes/(auth)/sign-up.tsx`、`web/src/routes/(auth)/register.tsx`、`web/src/routes/(auth)/forgot-password.tsx`、`web/src/routes/(auth)/reset.tsx`：企业版前端不展示注册、找回密码和重置密码页面，直达时重定向回登录页。
- `web/src/components/layout/components/app-header.tsx`、`web/src/components/layout/components/authenticated-layout.tsx`、`web/src/components/profile-dropdown.tsx`：隐藏顶部导航、搜索、公告、主题/语言、系统设置入口。
- `web/src/hooks/use-sidebar-data.ts`：左侧栏只保留 P0 需要的员工/管理员入口。
- `web/src/features/dashboard/index.tsx`、`web/src/features/dashboard/section-registry.tsx`：数据看板只保留模型调用分析和用户统计。
- `web/src/components/layout/components/public-header.tsx`、`web/src/components/layout/components/footer.tsx`：公开页兜底显示企业品牌，不再默认露出旧项目导航和署名。
- `web/src/features/channels/**`：渠道页保留功能，但隐藏跳转系统设置的辅助入口。

## 本地验证命令

在 `web/` 目录执行：

```bash
node node_modules/@typescript/native-preview/lib/tsgo.js -b
node node_modules/@rsbuild/core/bin/rsbuild.js build
```

说明：

- `tsgo -b` 用于确认 TypeScript 类型和路由类型可通过。
- `rsbuild build` 用于确认前端生产包可构建。
- 当前环境下 Vitest 的 Windows 执行入口和 worker 启动不稳定，本次未把单元测试作为上线阻断项；P0 以类型检查、生产构建和人工页面复核为准。

## 复核页面

上线后建议按下面顺序快速检查：

1. 退出登录后访问 `/`
   - 应进入简洁登录页。
   - 页面标题应为 `智颖API监控`。
   - 不应出现 New API logo、注册、忘记密码、第三方登录入口。
2. 登录后访问 `/`
   - 应进入 `/dashboard/models`。
3. 查看顶部栏
   - 不应出现主页、控制台、文档、关于、搜索框、通知、语言、主题设置。
   - 右侧头像菜单应保留退出登录。
   - 头像菜单不应出现系统设置。
4. 查看左侧栏
   - 普通用户不应看到管理员入口。
   - 管理员可看到渠道、用户。
   - 超管可额外看到系统信息。
   - 所有人都不应看到模型、兑换码、订阅、任务插件、系统设置。
5. 查看数据看板
   - 只应看到模型调用分析、用户统计。
   - 不应看到概览、分流。
6. 回归接口使用
   - 员工继续使用统一 Base URL。
   - 已有 API Key、用户、用量日志应不受影响。

## 回滚方式

如果上线后发现 UI 异常，直接回滚上一版 Docker 镜像即可。数据库无需回滚。

```bash
docker stop new-api-company
docker rm new-api-company
docker run ...上一版镜像...
```

回滚只影响前端展示和当前镜像代码，不影响 Postgres 数据卷。
