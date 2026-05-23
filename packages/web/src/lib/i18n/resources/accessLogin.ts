export const accessLoginEn = {
  eyebrow: 'Route Access',
  title: 'Sign in to continue',
  description:
    'This route is protected by Auth Gate. Sign in with a user account that has access to {{routeName}}.',
  submit: 'Continue to Route',
  fields: {
    username: 'Username',
    usernamePlaceholder: 'Enter username',
    password: 'Password',
    passwordPlaceholder: 'Enter password',
  },
  errors: {
    network: 'Network error',
  },
} as const

export const accessLoginZhCN = {
  eyebrow: '路由访问',
  title: '登录后继续',
  description: '该路由受 Auth Gate 保护。请使用有权访问 {{routeName}} 的账号登录。',
  submit: '继续访问路由',
  fields: {
    username: '用户名',
    usernamePlaceholder: '输入用户名',
    password: '密码',
    passwordPlaceholder: '输入密码',
  },
  errors: {
    network: '网络错误',
  },
} as const
