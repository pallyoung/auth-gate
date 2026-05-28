export const accessLoginEn = {
  eyebrow: 'Route Access',
  title: 'Sign in to continue',
  description:
    'This route is protected by Auth Gate. Sign in with a user account that has access to {{routeName}}.',
  protectedRouteFallback: 'Protected Route',
  submit: 'Continue to Route',
  fields: {
    username: 'Username',
    usernamePlaceholder: 'Enter username',
    password: 'Password',
    passwordPlaceholder: 'Enter password',
  },
  errors: {
    network: 'Network error',
    missingRoute: 'This access link is missing route details. Return to the protected app and try again.',
    routeUnavailable: 'This protected route is no longer available. Return to the app and request a fresh access link.',
    invalidCredentials: "We couldn't sign you in with that username and password. Check your credentials and try again.",
    routeAccessDenied: "This account doesn't have access to this protected route. Contact your administrator or try a different account.",
    userDisabled: 'This account has been disabled. Contact your administrator or try a different account.',
    sessionUnavailable:
      "We couldn't start your route access session right now. Try again in a moment.",
  },
} as const

export const accessLoginZhCN = {
  eyebrow: '路由访问',
  title: '登录后继续',
  description: '该路由受 Auth Gate 保护。请使用有权访问 {{routeName}} 的账号登录。',
  protectedRouteFallback: '受保护路由',
  submit: '继续访问路由',
  fields: {
    username: '用户名',
    usernamePlaceholder: '输入用户名',
    password: '密码',
    passwordPlaceholder: '输入密码',
  },
  errors: {
    network: '网络错误',
    missingRoute: '此访问链接缺少路由信息。请返回受保护应用后重试。',
    routeUnavailable: '该受保护路由已不可用。请返回应用并重新获取访问链接。',
    invalidCredentials: '该用户名和密码无法登录。请检查凭据后重试。',
    routeAccessDenied: '该账号无权访问这个受保护路由。请联系管理员或尝试其他账号。',
    userDisabled: '该账号已被禁用。请联系管理员或尝试其他账号。',
    sessionUnavailable: '当前无法启动路由访问会话。请稍后重试。',
  },
} as const
