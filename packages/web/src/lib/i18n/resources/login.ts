export const loginEn = {
  hero: {
    eyebrow: 'Secure Routing Workspace',
    title: 'Manage gateway traffic with a calmer, sharper control surface.',
    description:
      'Configure routes, apply authentication policies, and perform runtime operations from one focused console.',
    routingLabel: 'Routing',
    routingValue: 'Precise matching',
    authLabel: 'Auth',
    authValue: 'Policy per route',
    runtimeLabel: 'Runtime',
    runtimeValue: 'Safe reload operations',
  },
  card: {
    eyebrow: 'Auth Gate',
    title: 'Sign in to the control plane',
    description:
      'Use an administrator account to manage routes, authentication rules, and runtime operations.',
    submit: 'Sign In',
    sessionHint: 'Session access is scoped by your server-side permissions.',
  },
  fields: {
    username: 'Username',
    usernamePlaceholder: 'Enter username',
    password: 'Password',
    passwordPlaceholder: 'Enter password',
  },
  errors: {
    sessionExpired: 'Your session has expired. Sign in again to continue.',
    invalidCredentials:
      "We couldn't sign you in with that username and password. Check your credentials and try again.",
    userDisabled:
      'This account has been disabled. Contact your administrator or try a different account.',
    controlPlaneAccessDenied:
      "This account doesn't have access to the control plane. Contact your administrator or try a different account.",
    sessionUnavailable:
      "We couldn't start your control plane session right now. Try again in a moment.",
    network: 'Network error',
  },
} as const

export const loginZhCN = {
  hero: {
    eyebrow: '安全路由工作台',
    title: '用更冷静、更清晰的控制界面管理网关流量。',
    description: '在一个专注的控制台中配置路由、应用鉴权策略并执行运行时操作。',
    routingLabel: '路由',
    routingValue: '精准匹配',
    authLabel: '鉴权',
    authValue: '按路由设定策略',
    runtimeLabel: '运行时',
    runtimeValue: '安全热重载',
  },
  card: {
    eyebrow: 'Auth Gate',
    title: '登录控制台',
    description: '使用管理员账号管理路由、鉴权规则和运行时操作。',
    submit: '登录',
    sessionHint: '会话访问范围由服务端权限控制。',
  },
  fields: {
    username: '用户名',
    usernamePlaceholder: '输入用户名',
    password: '密码',
    passwordPlaceholder: '输入密码',
  },
  errors: {
    sessionExpired: '你的会话已过期。请重新登录后继续。',
    invalidCredentials: '无法使用该用户名和密码登录。请检查凭据后重试。',
    userDisabled: '该账号已被禁用。请联系管理员或尝试其他账号。',
    controlPlaneAccessDenied: '该账号无权访问控制台。请联系管理员或尝试其他账号。',
    sessionUnavailable: '当前无法启动控制台会话。请稍后重试。',
    network: '网络错误',
  },
} as const
