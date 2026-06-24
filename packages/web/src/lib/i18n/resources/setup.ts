export const setupEn = {
  hero: {
    eyebrow: 'First-time Setup',
    title: 'Create your admin account to get started.',
    description:
      'This is your first visit. Set up an administrator account to manage routes, authentication policies, and runtime operations.',
    step1Label: 'Step 1',
    step1Value: 'Create account',
    step2Label: 'Step 2',
    step2Value: 'Configure routes',
    step3Label: 'Step 3',
    step3Value: 'Go live',
  },
  card: {
    eyebrow: 'Auth Gate',
    title: 'Set up admin account',
    description:
      'Choose a username and password for the administrator account. You can add more users later.',
    submit: 'Create Admin Account',
  },
  fields: {
    username: 'Username',
    usernamePlaceholder: 'Enter username',
    password: 'Password',
    passwordPlaceholder: 'At least 8 characters',
    confirmPassword: 'Confirm Password',
    confirmPasswordPlaceholder: 'Re-enter password',
  },
  errors: {
    passwordTooShort: 'Password must be at least 8 characters.',
    passwordMismatch: 'Passwords do not match.',
    setupAlreadyCompleted:
      'An admin account already exists. Redirecting to login…',
    network: 'Network error. Please try again.',
  },
} as const

export const setupZhCN = {
  hero: {
    eyebrow: '首次配置',
    title: '创建管理员账号以开始使用。',
    description:
      '这是你的首次访问。请设置管理员账号来管理路由、鉴权策略和运行时操作。',
    step1Label: '第一步',
    step1Value: '创建账号',
    step2Label: '第二步',
    step2Value: '配置路由',
    step3Label: '第三步',
    step3Value: '上线运行',
  },
  card: {
    eyebrow: 'Auth Gate',
    title: '设置管理员账号',
    description: '选择用户名和密码创建管理员账号。后续可添加更多用户。',
    submit: '创建管理员账号',
  },
  fields: {
    username: '用户名',
    usernamePlaceholder: '输入用户名',
    password: '密码',
    passwordPlaceholder: '至少 8 个字符',
    confirmPassword: '确认密码',
    confirmPasswordPlaceholder: '再次输入密码',
  },
  errors: {
    passwordTooShort: '密码长度至少为 8 个字符。',
    passwordMismatch: '两次输入的密码不一致。',
    setupAlreadyCompleted: '管理员账号已存在，正在跳转到登录页…',
    network: '网络错误，请重试。',
  },
} as const
