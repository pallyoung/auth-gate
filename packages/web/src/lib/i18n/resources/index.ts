import { accessLoginEn, accessLoginZhCN } from './accessLogin'
import { accessLogsEn } from './accessLogs'
import { accessLogsZhCN } from './accessLogsZhCN'
import { authRulesEn, authRulesZhCN } from './authRules'
import { certificatesEn, certificatesZhCN } from './certificates'
import { commonEn, commonZhCN } from './common'
import { dashboardEn, dashboardZhCN } from './dashboard'
import { hostsEn, hostsZhCN } from './hosts'
import { layoutEn, layoutZhCN } from './layout'
import { loginEn, loginZhCN } from './login'
import { routesEn, routesZhCN } from './routes'
import { settingsEn, settingsZhCN } from './settings'
import { setupEn, setupZhCN } from './setup'
import { usersEn, usersZhCN } from './users'

export const resources = {
  en: {
    common: commonEn,
    layout: layoutEn,
    login: loginEn,
    setup: setupEn,
    accessLogin: accessLoginEn,
    accessLogs: accessLogsEn,
    dashboard: dashboardEn,
    routes: routesEn,
    authRules: authRulesEn,
    users: usersEn,
    certificates: certificatesEn,
    settings: settingsEn,
    hosts: hostsEn,
  },
  'zh-CN': {
    common: commonZhCN,
    layout: layoutZhCN,
    login: loginZhCN,
    setup: setupZhCN,
    accessLogin: accessLoginZhCN,
    accessLogs: accessLogsZhCN,
    dashboard: dashboardZhCN,
    routes: routesZhCN,
    authRules: authRulesZhCN,
    users: usersZhCN,
    certificates: certificatesZhCN,
    settings: settingsZhCN,
    hosts: hostsZhCN,
  },
} as const
