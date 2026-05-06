# Auth Gate Design System

## 1. 设计原则

- **一致性**: 统一的视觉语言，降低认知负担
- **清晰性**: 信息层次分明，操作路径清晰
- **高效性**: 减少操作步骤，提供快捷方式
- **可访问性**: 支持键盘导航，符合 WCAG 2.1 AA 标准

---

## 2. 色彩系统

### 2.1 主色调 (Primary)

用于主要操作、重要信息、品牌识别。

| Token | Light | Dark | 用途 |
|-------|-------|------|------|
| primary-50 | #eff6ff | #1e3a5f | 背景高亮 |
| primary-100 | #dbeafe | #1e4976 | 悬浮背景 |
| primary-200 | #bfdbfe | #1e5a8a | 边框 |
| primary-500 | #3b82f6 | #4299e1 | 主要按钮 |
| primary-600 | #2563eb | #4299e1 | 按钮悬浮 |
| primary-700 | #1d4ed8 | #3182ce | 按钮按下 |

### 2.2 中性色 (Neutral)

| Token | Light | Dark | 用途 |
|-------|-------|------|------|
| neutral-0 | #ffffff | #0f172a | 卡片背景 |
| neutral-50 | #f8fafc | #1e293b | 页面背景 |
| neutral-100 | #f1f5f9 | #334155 | 输入框背景 |
| neutral-200 | #e2e8f0 | #475569 | 分割线 |
| neutral-500 | #64748b | #94a3b8 | 次要文本 |
| neutral-700 | #334155 | #e2e8f0 | 标题文本 |
| neutral-900 | #0f172a | #f8fafc | 最高对比文本 |

### 2.3 语义色

| 类型 | Light | Dark | 用途 |
|------|-------|------|------|
| Success | #10b981 | #34d399 | 成功状态 |
| Warning | #f59e0b | #fbbf24 | 警告状态 |
| Error | #ef4444 | #f87171 | 错误状态 |
| Info | #3b82f6 | #60a5fa | 信息提示 |

---

## 3. 字体系统

字体栈: Inter (sans), JetBrains Mono (mono)

| Token | Size | Line Height | 用途 |
|-------|------|-------------|------|
| text-xs | 12px | 16px | 辅助文本 |
| text-sm | 14px | 20px | 常规文本 |
| text-base | 16px | 24px | 正文 |
| text-lg | 18px | 28px | 小标题 |
| text-xl | 20px | 28px | 标题 |
| text-2xl | 24px | 32px | 大标题 |
| text-3xl | 30px | 36px | 页面标题 |

---

## 4. 间距系统

基于 4px: 0, 4, 8, 12, 16, 20, 24, 32, 40, 48

---

## 5. 圆角系统

none(0), sm(4px), md(6px), lg(8px), xl(12px), full(9999px)

---

## 6. 阴影系统

sm, md, lg, xl (强度递增)

---

## 7. 动效

时长: fast(100ms), normal(200ms), slow(300ms)
缓动: ease-in, ease-out, ease-in-out

---

## 8. 组件规范

### 8.1 Button

尺寸: sm(32px), md(40px), lg(48px)
变体: primary, secondary, ghost, danger
状态: hover(亮度-10%), active(亮度-20%), disabled(透明度50%)

### 8.2 Input

尺寸: sm(32px), md(40px), lg(48px)
状态: default, hover, focus, error, disabled

### 8.3 Card

背景: bg-card
圆角: lg
阴影: md
内边距: 20-24px

### 8.4 Table

表头: 48px高
数据行: 56px高(紧凑40px)
状态: hover, selected

### 8.5 Badge

尺寸: sm(20px), md(24px)
变体: default, primary, success, warning, error

### 8.6 Switch

尺寸: sm(36x20), md(44x24)
状态: off(neutral-300), on(primary-500)

### 8.7 Toast

位置: 右上角
动画: 滑入
自动关闭: 4秒
变体: success, error, warning, info

### 8.8 Modal

尺寸: sm(400px), md(560px), lg(720px)
遮罩: 黑色50%
动画: 淡入+缩放

### 8.9 Empty State

图标: 48-64px, neutral-400
标题: 16-18px, neutral-700
描述: 14px, neutral-500

---

## 9. 图标

Lucide Icons, 尺寸: 16, 18, 20, 24px

---

## 10. 响应式

sm(640), md(768), lg(1024), xl(1280), 2xl(1536)

---

## 11. 暗色模式

CSS 变量 + localStorage 存储

---

## 12. 无障碍

键盘导航、focus ring、对比度>=4.5:1、aria-label
