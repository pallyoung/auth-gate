# Auth Gate Design System

## 1. 设计方向

- **Control Plane Aesthetic**: 面向基础设施后台，强调“精密、稳定、可审计”。
- **Warm Tech Contrast**: 使用暖灰背景承托青绿色主品牌，避免千篇一律的冷蓝后台。
- **Layered Surfaces**: 大量使用玻璃化面板、柔和描边和层级阴影，提升信息分区感。
- **Operational Clarity**: 所有关键动作、状态、风险提示都必须在视觉上足够明确。

---

## 2. 色彩系统

### 2.1 品牌主色

| Token | Light | Dark | 用途 |
|-------|-------|------|------|
| primary-100 | #c8f4ee | #14484b | 浅色强调背景 |
| primary-300 | #61d0c4 | #1d7f81 | 图形渐变过渡 |
| primary-500 | #0f8f8b | #38c7ba | 主按钮、焦点色、品牌识别 |
| primary-700 | #105e60 | #99ebe1 | 深色文字强调、渐变终点 |

### 2.2 辅助强调色

| Token | Light | Dark | 用途 |
|-------|-------|------|------|
| accent-100 | #fde9c0 | #50361a | 运营提示背景 |
| accent-300 | #efbb4d | #8f6327 | 数据卡强调色 |
| accent-500 | #bd7a18 | #d6a43d | 暖色辅助状态 |

### 2.3 中性色

| Token | Light | Dark | 用途 |
|-------|-------|------|------|
| neutral-0 | #ffffff | #111a25 | 卡片基底 |
| neutral-50 | #f6f3ec | #182230 | 页面底色 |
| neutral-100 | #ece7dc | #223043 | 分层背景 |
| neutral-500 | #6b6256 | #9ca9be | 次级文本 |
| neutral-900 | #17212d | #f8fafc | 主文本 |

### 2.4 语义色

| 类型 | Light | Dark | 用途 |
|------|-------|------|------|
| Success | #1f9d67 | #44cf8e | 正常运行、成功反馈 |
| Warning | #c9811d | #ebb15b | 配置提醒、风险提示 |
| Error | #d0474b | #ff7f84 | 错误、危险操作 |
| Info | #2f72c8 | #70a7ff | 普通说明、辅助信息 |

---

## 3. 字体系统

- **Sans**: `Plus Jakarta Sans`
- **Display**: `Space Grotesk`
- **Mono**: `IBM Plex Mono`

| Token | Size | Line Height | 用途 |
|-------|------|-------------|------|
| text-xs | 12px | 16px | 标签、说明 |
| text-sm | 14px | 20px | 表格、控件、辅助文案 |
| text-base | 16px | 24px | 正文 |
| text-lg | 18px | 28px | 卡片标题 |
| text-xl | 22px | 30px | 分区标题 |
| text-2xl | 30px | 38px | 页面标题 |
| text-3xl | 42px | 48px | 登录页大标题 |

规则：
- 页面标题和关键指标优先使用 `font-display`
- 表单、表格正文使用 `font-sans`
- URL、路径、后端地址、配置键使用 `font-mono`

---

## 4. 空间与圆角

### 4.1 间距

基于 4px 体系扩展：4, 8, 12, 16, 20, 24, 28, 32, 40, 48, 64

### 4.2 圆角

| Token | 值 | 用途 |
|------|----|------|
| radius-sm | 8px | badge、小型辅助块 |
| radius-md | 14px | 输入框、按钮 |
| radius-lg | 20px | 卡片、表格容器 |
| radius-xl | 28px | 页面头部、弹窗 |
| radius-2xl | 36px | 登录页、大型容器 |
| radius-full | 9999px | 胶囊按钮、状态标签 |

---

## 5. 阴影与材质

### 5.1 阴影

- `shadow-sm`: 轻量浮层、输入框
- `shadow-md`: 默认卡片、侧边导航
- `shadow-lg`: 强调卡片、主按钮 hover
- `shadow-xl`: 模态框、关键视觉块

### 5.2 材质

- `glass-panel`: 标准玻璃卡片
- `glass-panel-strong`: 高密度玻璃面板，用于侧边栏和弹窗
- `bg-card-soft`: 弱对比容器背景
- `bg-elevated`: 强对比浮层背景

---

## 6. 动效

- `duration-fast`: 140ms
- `duration-normal`: 220ms
- `duration-slow`: 360ms
- `ease-out`: 控件 hover / entrance
- `ease-in-out`: 脉冲、状态切换

关键动画：
- `animate-rise-in`: 页面内容轻微上浮进入
- `animate-modal-enter`: 弹窗淡入 + 上移缩放
- `animate-pulse-glow`: 品牌点位与重点状态脉冲

---

## 7. 布局规范

### 7.1 应用壳

- Desktop 使用固定左侧玻璃导航栏
- Mobile 使用顶部栏 + 底部胶囊导航
- 主内容区域最大宽度 `1240px`

### 7.2 页面结构

推荐顺序：
1. `PageHeader`
2. `Alert`（如有）
3. `MetricCard` 概览区
4. 主内容卡片 / 表格 / 设置面板

---

## 8. 组件规范

### 8.1 Button

- 变体：`primary`, `secondary`, `ghost`, `danger`
- 形态：圆角胶囊，强调 hover 位移与阴影变化
- `primary` 使用青绿色渐变

### 8.2 Input / Select

- 高度：40 / 48 / 56
- 形态：18px 圆角，半透明玻璃输入面
- 标签统一使用大写小字号说明风格

### 8.3 Card

- 默认使用玻璃材质
- `tone`: `default`, `soft`, `strong`, `inverse`
- 信息分区必须通过卡片层次表达，不直接堆纯文本

### 8.4 Badge

- 使用全大写胶囊标签
- 用于状态、系统分类、页面 meta

### 8.5 Table

- 头部采用轻渐变 + 大写栏目标题
- 行 hover 仅轻量变色，不做激进高亮
- 移动端切换为卡片列表

### 8.6 Switch

- 不使用裸露 toggle
- 每个开关包含标签与说明，放入带边框的独立容器

### 8.7 Modal

- 大圆角底部抽屉式移动端体验
- Desktop 居中玻璃面板
- 顶部 sticky 标题栏 + 关闭按钮

### 8.8 Alert

- 变体：`info`, `success`, `warning`, `error`
- 用于页面级反馈，优先于浏览器原生提示样式

### 8.9 MetricCard

- 页面概览组件
- 包含 `label`, `value`, `hint`, `icon`
- 页面首屏推荐 3 张以内

---

## 9. 图标

- 图标库：`lucide-react`
- 常用尺寸：16, 20, 24
- 图标通常置于柔和底色容器中，不直接裸放

---

## 10. 响应式

- `sm`: 640
- `md`: 768
- `lg`: 1024
- `xl`: 1280

策略：
- 表格在 `md` 以下转为卡片
- 关键操作按钮在移动端允许占满宽度
- 登录页品牌大视觉仅在 `lg` 及以上显示

---

## 11. 无障碍

- 所有交互控件保留 focus ring
- 按钮、图标按钮、导航项必须具备可见 hover/focus 状态
- 错误与成功提示使用语义色并保留文本说明
- 弹窗支持 Escape 关闭与焦点回退

---

## 12. 当前组件库清单

- `Button`
- `Input`
- `Select`
- `Switch`
- `Card`
- `Badge`
- `Table`
- `Modal`
- `EmptyState`
- `Alert`
- `MetricCard`
- `PageHeader`

这套组件库服务于 Auth Gate 当前三类主要页面：路由管理、鉴权规则管理、运行时设置。
