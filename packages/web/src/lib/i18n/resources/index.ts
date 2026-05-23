import { accessLoginEn, accessLoginZhCN } from './accessLogin'
import { commonEn, commonZhCN } from './common'
import { layoutEn, layoutZhCN } from './layout'
import { loginEn, loginZhCN } from './login'

export const resources = {
  en: {
    common: commonEn,
    layout: layoutEn,
    login: loginEn,
    accessLogin: accessLoginEn,
  },
  'zh-CN': {
    common: commonZhCN,
    layout: layoutZhCN,
    login: loginZhCN,
    accessLogin: accessLoginZhCN,
  },
} as const
