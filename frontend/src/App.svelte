<script lang="ts">
  import './app.css'

  type User = { id: number; email: string; nickname?: string; emailVerified: boolean; role: string }
  type Credential = { companyCode: string; employeeAccount: string; status: string; lastLoginAt?: string } | null
  type ScheduleSegment = { name?: string; startTime?: string; endTime?: string; hours?: number; classCode?: string }
  type Schedule = { id: number; shiftDate: string; shiftName?: string; startTime?: string; endTime?: string; hours?: number; classCode?: string; segments?: ScheduleSegment[] }
  type SyncRun = { id: number; startedAt: string; finishedAt?: string; status: string; errorMessage?: string; entryCount: number; marked?: boolean }
  type CalendarSubscription = { url: string; webcalUrl: string } | null
  type CalendarRequestLog = { id: number; requestedAt: string; userAgent: string; remoteAddr: string; path: string }
  type AdminUser = { id: number; email: string; nickname?: string; emailVerified: boolean; role: string; createdAt: string; lastLoginAt?: string }
  type Locale = 'zh-HK' | 'en'
  type Theme = 'light' | 'dark'
  type AuthMode = 'login' | 'register' | 'verify' | 'forgot' | 'reset'
  type PublicConfig = { emailVerificationRequired: boolean }

  const messages = {
    'zh-HK': {
      appName: 'Gaia Calendar',
      authTitle: '控制台',
      authSubtitle: '同步 Gaia 排班、轉換 Calendar 訂閱。',
      login: '登入',
      register: '註冊',
      verify: '輸入驗證碼',
      forgot: '忘記密碼',
      resetPassword: '重設密碼',
      email: 'Email',
      loginIdentifier: 'Email / Nickname',
      nickname: 'Nickname',
      password: '密碼',
      confirmPassword: '確認密碼',
      newPassword: '新密碼',
      passwordMismatch: '兩次輸入的密碼不一致。',
      verificationCode: 'Email 驗證碼',
      loginAction: '登入',
      createAccount: '建立帳號',
      verifyEmail: '驗證 Email',
      sendReset: '寄出重設連結',
      updatePassword: '修改密碼',
      backToLogin: '返回登入',
      alreadyHaveCode: '已有驗證碼',
      resetHint: '輸入註冊 Email，我會寄出此服務網域的一次性重設連結。',
      resetTokenMissing: '重設連結缺少 token，請重新申請。',
      codeSent: '驗證碼已寄出，請檢查你的 Email。',
      accountCreated: '帳號已建立，可以登入了。',
      emailVerified: 'Email 已驗證，可以登入了。',
      resetSent: '如果帳號存在，重設連結會寄到你的 Email。',
      passwordReset: '密碼已修改，請重新登入。',
      verified: '已驗證',
      logout: '登出',
      mySchedule: '我的排班',
      company: '公司',
      gaiaNotSet: '尚未設定 Gaia 帳號',
      sync: '同步',
      gaiaCredential: 'Gaia 登入資料',
      companyCode: '公司代碼',
      employeeAccount: '員工帳號',
      gaiaPassword: 'Gaia 密碼',
      saveCredential: '保存登入資料',
      status: '狀態',
      notConfigured: '未設定',
      syncHistory: '同步記錄',
      calendarSubscription: 'Calendar 訂閱',
      calendarHint: 'iPhone 可直接訂閱這個服務的 Calendar feed；伺服器會定時同步 Gaia 排班。',
      subscribeIphone: 'iPhone 訂閱',
      copyUrl: '複製 URL',
      rotateUrl: '重置 URL',
      calendarRequestLogs: 'ICS 更新記錄',
      lastCalendarRequest: '最後請求',
      noCalendarRequests: '還沒有 Calendar app 請求記錄。',
      unknownClient: '未知客戶端',
      adminPanel: '使用者管理',
      totalUsers: '使用者總數',
      role: '權限',
      emailStatus: 'Email 狀態',
      verifiedEmail: '已驗證',
      unverifiedEmail: '未驗證',
      lastLogin: '最後登入',
      createdAt: '建立時間',
      noUsers: '沒有使用者資料。',
      updatedUser: '使用者已更新。',
      copied: '已複製訂閱 URL。',
      rotated: '訂閱 URL 已重置，舊 URL 已失效。',
      refresh: '刷新',
      mark: '標記',
      unmark: '取消標記',
      deleteRun: '刪除',
      totalHours: '預計總工時',
      noRuns: '還沒有同步記錄。',
      roster: '排班',
      noSchedule: '這個月份沒有排班資料。',
      noScheduledShift: '無排班',
      scheduledShift: '排班',
      details: '明細',
      language: '語言',
      theme: '主題',
      light: '淺色',
      dark: '深色',
      savedCredential: 'Gaia 登入資料已保存。',
      syncedPrefix: '已同步',
      syncedSuffix: '筆排班。'
    },
    en: {
      appName: 'Gaia Calendar',
      authTitle: 'Schedule Control',
      authSubtitle: 'Sync Gaia rosters, save credentials, and review shifts in one place.',
      login: 'Login',
      register: 'Register',
      verify: 'Enter code',
      forgot: 'Forgot password',
      resetPassword: 'Reset password',
      email: 'Email',
      loginIdentifier: 'Email / Nickname',
      nickname: 'Nickname',
      password: 'Password',
      confirmPassword: 'Confirm password',
      newPassword: 'New password',
      passwordMismatch: 'Passwords do not match.',
      verificationCode: 'Email verification code',
      loginAction: 'Login',
      createAccount: 'Create account',
      verifyEmail: 'Verify email',
      sendReset: 'Send reset link',
      updatePassword: 'Update password',
      backToLogin: 'Back to login',
      alreadyHaveCode: 'Already have a code',
      resetHint: 'Enter your registered email and a one-time reset link for this service will be sent.',
      resetTokenMissing: 'This reset link is missing a token. Request a new one.',
      codeSent: 'Verification code sent. Check your email.',
      accountCreated: 'Account created. You can log in now.',
      emailVerified: 'Email verified. You can log in now.',
      resetSent: 'If the account exists, a reset link will be sent to your email.',
      passwordReset: 'Password updated. Please log in again.',
      verified: 'Verified',
      logout: 'Logout',
      mySchedule: 'My Schedule',
      company: 'Company',
      gaiaNotSet: 'Gaia account not set',
      sync: 'Sync',
      gaiaCredential: 'Gaia credential',
      companyCode: 'Company code',
      employeeAccount: 'Employee account',
      gaiaPassword: 'Gaia password',
      saveCredential: 'Save credential',
      status: 'Status',
      notConfigured: 'not configured',
      syncHistory: 'Sync history',
      calendarSubscription: 'Calendar subscription',
      calendarHint: 'iPhone can subscribe directly to this service Calendar feed. The server syncs Gaia rosters on a schedule.',
      subscribeIphone: 'Subscribe on iPhone',
      copyUrl: 'Copy URL',
      rotateUrl: 'Reset URL',
      calendarRequestLogs: 'ICS update log',
      lastCalendarRequest: 'Last request',
      noCalendarRequests: 'No Calendar app request logs yet.',
      unknownClient: 'Unknown client',
      adminPanel: 'User management',
      totalUsers: 'Total users',
      role: 'Role',
      emailStatus: 'Email status',
      verifiedEmail: 'Verified',
      unverifiedEmail: 'Unverified',
      lastLogin: 'Last login',
      createdAt: 'Created',
      noUsers: 'No users found.',
      updatedUser: 'User updated.',
      copied: 'Subscription URL copied.',
      rotated: 'Subscription URL reset. The old URL is invalid.',
      refresh: 'Refresh',
      mark: 'Mark',
      unmark: 'Unmark',
      deleteRun: 'Delete',
      totalHours: 'Projected hours',
      noRuns: 'No sync runs yet.',
      roster: 'roster',
      noSchedule: 'No schedule data for this month.',
      noScheduledShift: 'No scheduled shift',
      scheduledShift: 'Scheduled shift',
      details: 'Details',
      language: 'Language',
      theme: 'Theme',
      light: 'Light',
      dark: 'Dark',
      savedCredential: 'Gaia credential saved.',
      syncedPrefix: 'Synced',
      syncedSuffix: 'schedule entries.'
    }
  } as const

  type MessageKey = keyof (typeof messages)['zh-HK']

  const savedLocale = localStorage.getItem('locale')
  const savedTheme = localStorage.getItem('theme')

  let locale: Locale = savedLocale === 'en' || savedLocale === 'zh-HK' ? savedLocale : 'zh-HK'
  let theme: Theme = savedTheme === 'light' || savedTheme === 'dark' ? savedTheme : 'dark'
  let user: User | null = null
  let credential: Credential = null
  let calendarSubscription: CalendarSubscription = null
  let calendarRequestLogs: CalendarRequestLog[] = []
  let adminUsers: AdminUser[] = []
  let adminUserTotal = 0
  let schedules: Schedule[] = []
  let runs: SyncRun[] = []
  let totalHours = 0
  let publicConfig: PublicConfig = { emailVerificationRequired: true }
  let mode: AuthMode = 'login'
  let message = ''
  let loading = false
  let month = new Date().toISOString().slice(0, 7)
  let resetToken = ''

  let email = ''
  let nickname = ''
  let password = ''
  let confirmPassword = ''
  let newPassword = ''
  let code = ''
  let companyCode = ''
  let employeeAccount = ''
  let gaiaPassword = ''

  const resetParams = new URLSearchParams(window.location.search)
  if (window.location.pathname === '/reset-password') {
    mode = 'reset'
    resetToken = resetParams.get('token') || ''
    if (!resetToken) message = tr('resetTokenMissing')
  }

  function tr(key: MessageKey) {
    return messages[locale][key]
  }

  function setLocale(next: Locale) {
    locale = next
    localStorage.setItem('locale', next)
  }

  function setTheme(next: Theme) {
    theme = next
    localStorage.setItem('theme', next)
    document.documentElement.dataset.theme = next
  }

  setTheme(theme)

  async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
    const res = await fetch(path, {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
      ...options
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || res.statusText)
    return data
  }

  async function loadPublicConfig() {
    publicConfig = await api<PublicConfig>('/api/config')
  }

  async function loadMe() {
    try {
      const data = await api<{ user: User }>('/api/me')
      user = data.user
      await Promise.all([loadCredential(), loadCalendarSubscription(), loadCalendarRequestLogs(), loadSchedules(), loadRuns()])
      if (user.role === 'admin') await loadAdminUsers()
    } catch {
      user = null
    }
  }

  async function register() {
    loading = true
    message = ''
    try {
      if (password !== confirmPassword) {
        message = tr('passwordMismatch')
        return
      }
      await api('/api/auth/register', { method: 'POST', body: JSON.stringify({ email, nickname, password, confirmPassword, locale }) })
      if (publicConfig.emailVerificationRequired) {
        mode = 'verify'
        message = tr('codeSent')
      } else {
        mode = 'login'
        message = tr('accountCreated')
      }
    } catch (err) {
      message = String((err as Error).message)
    } finally {
      loading = false
    }
  }

  async function verify() {
    loading = true
    message = ''
    try {
      await api('/api/auth/verify', { method: 'POST', body: JSON.stringify({ email, code }) })
      mode = 'login'
      message = tr('emailVerified')
    } catch (err) {
      message = String((err as Error).message)
    } finally {
      loading = false
    }
  }

  async function requestPasswordReset() {
    loading = true
    message = ''
    try {
      await api('/api/auth/request-password-reset', { method: 'POST', body: JSON.stringify({ email, locale }) })
      mode = 'login'
      message = tr('resetSent')
    } catch (err) {
      message = String((err as Error).message)
    } finally {
      loading = false
    }
  }

  async function resetPassword() {
    loading = true
    message = ''
    try {
      await api('/api/auth/reset-password', { method: 'POST', body: JSON.stringify({ token: resetToken, newPassword }) })
      mode = 'login'
      password = ''
      confirmPassword = ''
      newPassword = ''
      message = tr('passwordReset')
      history.replaceState(null, '', '/')
    } catch (err) {
      message = String((err as Error).message)
    } finally {
      loading = false
    }
  }

  async function login() {
    loading = true
    message = ''
    try {
      const data = await api<{ user: User }>('/api/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) })
      user = data.user
      await Promise.all([loadCredential(), loadCalendarSubscription(), loadCalendarRequestLogs(), loadSchedules(), loadRuns()])
      if (user.role === 'admin') await loadAdminUsers()
    } catch (err) {
      message = String((err as Error).message)
    } finally {
      loading = false
    }
  }

  async function logout() {
    await api('/api/auth/logout', { method: 'POST' })
    user = null
    credential = null
    calendarSubscription = null
    calendarRequestLogs = []
    adminUsers = []
    adminUserTotal = 0
    schedules = []
    runs = []
    totalHours = 0
  }

  async function loadCredential() {
    const data = await api<{ credential: Credential }>('/api/gaia-credential')
    credential = data.credential
    if (credential) {
      companyCode = credential.companyCode
      employeeAccount = credential.employeeAccount
    }
  }

  async function saveCredential() {
    loading = true
    message = ''
    try {
      await api('/api/gaia-credential', { method: 'PUT', body: JSON.stringify({ companyCode, employeeAccount, password: gaiaPassword }) })
      gaiaPassword = ''
      await loadCredential()
      message = tr('savedCredential')
    } catch (err) {
      message = String((err as Error).message)
    } finally {
      loading = false
    }
  }

  async function syncSchedules() {
    loading = true
    message = ''
    try {
      const data = await api<{ entryCount: number }>('/api/schedules/sync', { method: 'POST' })
      await Promise.all([loadSchedules(), loadRuns(), loadCredential()])
      message = `${tr('syncedPrefix')} ${data.entryCount} ${tr('syncedSuffix')}`
    } catch (err) {
      message = String((err as Error).message)
      await loadRuns()
    } finally {
      loading = false
    }
  }

  async function loadSchedules() {
    const data = await api<{ schedules: Schedule[]; totalHours: number }>(`/api/schedules?month=${month}`)
    schedules = data.schedules
    totalHours = data.totalHours || 0
  }

  async function loadRuns() {
    const data = await api<{ runs: SyncRun[] }>('/api/sync-runs')
    runs = data.runs
  }

  async function loadCalendarSubscription() {
    const data = await api<CalendarSubscription>('/api/calendar-subscription')
    calendarSubscription = data
  }

  async function loadCalendarRequestLogs() {
    const data = await api<{ logs: CalendarRequestLog[] }>('/api/calendar-request-logs')
    calendarRequestLogs = data.logs
  }

  async function loadAdminUsers() {
    const data = await api<{ total: number; users: AdminUser[] }>('/api/admin/users')
    adminUserTotal = data.total
    adminUsers = data.users
  }

  async function rotateCalendarSubscription() {
    loading = true
    message = ''
    try {
      calendarSubscription = await api<CalendarSubscription>('/api/calendar-subscription/rotate', { method: 'POST' })
      await loadCalendarRequestLogs()
      message = tr('rotated')
    } catch (err) {
      message = String((err as Error).message)
    } finally {
      loading = false
    }
  }

  async function copyCalendarUrl() {
    if (!calendarSubscription?.url) return
    await navigator.clipboard.writeText(calendarSubscription.url)
    message = tr('copied')
  }

  function subscribeOnIphone() {
    if (!calendarSubscription?.webcalUrl) return
    window.location.href = calendarSubscription.webcalUrl
  }

  async function setRunMarked(run: SyncRun, marked: boolean) {
    await api(`/api/sync-runs/${run.id}`, { method: 'PATCH', body: JSON.stringify({ marked }) })
    await loadRuns()
  }

  async function deleteRun(run: SyncRun) {
    await api(`/api/sync-runs/${run.id}`, { method: 'DELETE' })
    await loadRuns()
  }

  async function updateAdminUser(target: AdminUser, patch: Partial<Pick<AdminUser, 'role' | 'nickname' | 'emailVerified'>>) {
    const data = await api<{ user: AdminUser }>(`/api/admin/users/${target.id}`, { method: 'PATCH', body: JSON.stringify(patch) })
    adminUsers = adminUsers.map((item) => (item.id === target.id ? data.user : item))
    message = tr('updatedUser')
  }

  function fmtTime(value?: string) {
    if (!value) return ''
    const match = value.match(/T(\d{2}:\d{2})/)
    if (match) return match[1]
    return value.slice(0, 5)
  }

  function fmtHours(value?: number) {
    if (value == null || value < 0) return ''
    return `${Number.isInteger(value) ? value : value.toFixed(1)}h`
  }

  function shiftLabel(item: Schedule) {
    if (item.classCode === 'no_schedule') return tr('noScheduledShift')
    return item.shiftName || tr('scheduledShift')
  }

  function scheduleMeta(item: Schedule) {
    if (item.classCode === 'no_schedule') return ''
    return fmtHours(item.hours) || item.classCode || ''
  }

  function segmentLabel(segment: ScheduleSegment) {
    return segment.name || segment.classCode || tr('details')
  }

  function segmentMeta(segment: ScheduleSegment) {
    const time = segment.startTime ? `${segment.startTime}${segment.endTime ? ` - ${segment.endTime}` : ''}` : ''
    const hours = fmtHours(segment.hours)
    return [time, hours].filter(Boolean).join(' · ')
  }

  function calendarClient(log: CalendarRequestLog) {
    return log.userAgent || tr('unknownClient')
  }

  async function init() {
    await loadPublicConfig().catch(() => {
      publicConfig = { emailVerificationRequired: true }
    })
    await loadMe()
  }

  init()
</script>

{#if !user}
  <main class="auth-shell">
    <section class="auth-panel">
      <div class="topbar compact">
        <p class="eyebrow">{tr('appName')}</p>
        <div class="prefs">
          <select aria-label={tr('language')} bind:value={locale} on:change={() => setLocale(locale)}>
            <option value="zh-HK">繁中</option>
            <option value="en">EN</option>
          </select>
          <button class="ghost small" on:click={() => setTheme(theme === 'dark' ? 'light' : 'dark')}>{theme === 'dark' ? tr('light') : tr('dark')}</button>
        </div>
      </div>

      <div>
        <h1>{mode === 'reset' ? tr('resetPassword') : tr('authTitle')}</h1>
        <p class="muted">{mode === 'forgot' ? tr('resetHint') : tr('authSubtitle')}</p>
      </div>

      {#if mode !== 'reset'}
        <div class="tabs two-tabs">
          <button class:active={mode === 'login'} on:click={() => (mode = 'login')}>{tr('login')}</button>
          <button class:active={mode === 'register' || mode === 'verify'} on:click={() => (mode = 'register')}>{tr('register')}</button>
        </div>
      {/if}

      {#if mode !== 'reset'}
        <label>{mode === 'login' ? tr('loginIdentifier') : tr('email')}<input bind:value={email} type={mode === 'login' ? 'text' : 'email'} autocomplete={mode === 'login' ? 'username' : 'email'} /></label>
      {/if}
      {#if mode === 'login' || mode === 'register'}
        {#if mode === 'register'}
          <label>{tr('nickname')}<input bind:value={nickname} autocomplete="username" /></label>
        {/if}
        <label>{tr('password')}<input bind:value={password} type="password" autocomplete={mode === 'login' ? 'current-password' : 'new-password'} /></label>
        {#if mode === 'register'}
          <label>{tr('confirmPassword')}<input bind:value={confirmPassword} type="password" autocomplete="new-password" /></label>
        {/if}
      {:else if mode === 'verify'}
        <label>{tr('verificationCode')}<input bind:value={code} inputmode="numeric" /></label>
      {:else if mode === 'reset'}
        <label>{tr('newPassword')}<input bind:value={newPassword} type="password" autocomplete="new-password" /></label>
      {/if}

      {#if mode === 'login'}
        <button class="primary" disabled={loading} on:click={login}>{tr('loginAction')}</button>
        <button class="link-button" on:click={() => (mode = 'forgot')}>{tr('forgot')}</button>
      {:else if mode === 'register'}
        <button class="primary" disabled={loading} on:click={register}>{tr('createAccount')}</button>
        {#if publicConfig.emailVerificationRequired}
          <button class="link-button" on:click={() => (mode = 'verify')}>{tr('alreadyHaveCode')}</button>
        {/if}
      {:else if mode === 'verify'}
        <button class="primary" disabled={loading} on:click={verify}>{tr('verifyEmail')}</button>
        <button class="link-button" on:click={() => (mode = 'register')}>{tr('register')}</button>
      {:else if mode === 'forgot'}
        <button class="primary" disabled={loading} on:click={requestPasswordReset}>{tr('sendReset')}</button>
        <button class="link-button" on:click={() => (mode = 'login')}>{tr('backToLogin')}</button>
      {:else}
        <button class="primary" disabled={loading || !resetToken} on:click={resetPassword}>{tr('updatePassword')}</button>
        <button class="link-button" on:click={() => (mode = 'login')}>{tr('backToLogin')}</button>
      {/if}

      {#if message}<p class="notice">{message}</p>{/if}
    </section>
  </main>
{:else}
  <main class="app-shell">
    <aside>
      <div class="side-head">
        <p class="eyebrow">{tr('appName')}</p>
        <div class="prefs">
          <select aria-label={tr('language')} bind:value={locale} on:change={() => setLocale(locale)}>
            <option value="zh-HK">繁中</option>
            <option value="en">EN</option>
          </select>
          <button class="ghost small" on:click={() => setTheme(theme === 'dark' ? 'light' : 'dark')}>{theme === 'dark' ? tr('light') : tr('dark')}</button>
        </div>
      </div>
      <h2>{user.email}</h2>
      <p class="status ok">{tr('verified')}</p>
      <button class="ghost" on:click={logout}>{tr('logout')}</button>
    </aside>

    <section class="workspace">
      <header>
        <div>
          <h1>{tr('mySchedule')}</h1>
          <p class="muted">{tr('company')} {credential?.companyCode || companyCode} · {credential?.employeeAccount || tr('gaiaNotSet')}</p>
        </div>
        <div class="actions">
          <input bind:value={month} type="month" on:change={loadSchedules} />
          <button class="primary" disabled={loading || !credential} on:click={syncSchedules}>{tr('sync')}</button>
        </div>
      </header>

      {#if message}<p class="notice">{message}</p>{/if}

      <section class="grid two">
        <div class="panel">
          <h3>{tr('gaiaCredential')}</h3>
          <label>{tr('companyCode')}<input bind:value={companyCode} /></label>
          <label>{tr('employeeAccount')}<input bind:value={employeeAccount} /></label>
          <label>{tr('gaiaPassword')}<input bind:value={gaiaPassword} type="password" /></label>
          <button class="primary" disabled={loading} on:click={saveCredential}>{tr('saveCredential')}</button>
          <p class="muted">{tr('status')}: {credential?.status || tr('notConfigured')}</p>
        </div>

        <div class="panel">
          <h3>{tr('calendarSubscription')}</h3>
          <p class="muted">{tr('calendarHint')}</p>
          <input readonly value={calendarSubscription?.url || ''} />
          <div class="button-row">
            <button class="primary" disabled={!calendarSubscription} on:click={subscribeOnIphone}>{tr('subscribeIphone')}</button>
            <button class="ghost" disabled={!calendarSubscription} on:click={copyCalendarUrl}>{tr('copyUrl')}</button>
            <button class="ghost" disabled={loading} on:click={rotateCalendarSubscription}>{tr('rotateUrl')}</button>
          </div>
          <div class="panel-title-row tight">
            <h4>{tr('calendarRequestLogs')}</h4>
            <button class="ghost small" on:click={loadCalendarRequestLogs}>{tr('refresh')}</button>
          </div>
          {#if calendarRequestLogs.length === 0}
            <p class="muted">{tr('noCalendarRequests')}</p>
          {:else}
            <div class="request-log-list">
              <div class="latest-request">
                <span>{tr('lastCalendarRequest')}</span>
                <strong>{new Date(calendarRequestLogs[0].requestedAt).toLocaleString(locale)}</strong>
              </div>
              {#each calendarRequestLogs.slice(0, 5) as log}
                <article>
                  <strong>{calendarClient(log)}</strong>
                  <span>{new Date(log.requestedAt).toLocaleString(locale)}</span>
                  {#if log.remoteAddr}<small>{log.remoteAddr}</small>{/if}
                </article>
              {/each}
            </div>
          {/if}
        </div>

        <div class="panel">
          <div class="panel-title-row">
            <h3>{tr('syncHistory')}</h3>
            <button class="ghost small" on:click={loadRuns}>{tr('refresh')}</button>
          </div>
          {#if runs.length === 0}
            <p class="muted">{tr('noRuns')}</p>
          {:else}
            <div class="run-list">
              {#each runs.slice(0, 5) as run}
                <div class:marked={run.marked}>
                  <div class="run-head">
                    <strong>{run.status}</strong>
                    <span>{new Date(run.startedAt).toLocaleString(locale)}</span>
                  </div>
                  {#if run.errorMessage}<small>{run.errorMessage}</small>{/if}
                  <div class="button-row compact">
                    <button class="ghost small" on:click={() => setRunMarked(run, !run.marked)}>{run.marked ? tr('unmark') : tr('mark')}</button>
                    <button class="ghost small danger" on:click={() => deleteRun(run)}>{tr('deleteRun')}</button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        {#if user.role === 'admin'}
          <div class="panel">
            <div class="panel-title-row">
              <h3>{tr('adminPanel')}</h3>
              <button class="ghost small" on:click={loadAdminUsers}>{tr('refresh')}</button>
            </div>
            <div class="summary-strip">
              <span>{tr('totalUsers')}</span>
              <strong>{adminUserTotal}</strong>
            </div>
            {#if adminUsers.length === 0}
              <p class="muted">{tr('noUsers')}</p>
            {:else}
              <div class="admin-user-list">
                {#each adminUsers as adminUser}
                  <article>
                    <div>
                      <strong>{adminUser.email}</strong>
                      <span>{tr('nickname')}: {adminUser.nickname || '-'}</span>
                      <span>{tr('createdAt')}: {new Date(adminUser.createdAt).toLocaleString(locale)}</span>
                      <span>{tr('lastLogin')}: {adminUser.lastLoginAt ? new Date(adminUser.lastLoginAt).toLocaleString(locale) : '-'}</span>
                    </div>
                    <label>{tr('nickname')}
                      <input value={adminUser.nickname || ''} on:change={(event) => updateAdminUser(adminUser, { nickname: (event.currentTarget as HTMLInputElement).value })} />
                    </label>
                    <label>{tr('role')}
                      <select value={adminUser.role} on:change={(event) => updateAdminUser(adminUser, { role: (event.currentTarget as HTMLSelectElement).value })}>
                        <option value="user">user</option>
                        <option value="admin">admin</option>
                      </select>
                    </label>
                    <label class="checkbox-row">
                      <input type="checkbox" checked={adminUser.emailVerified} on:change={(event) => updateAdminUser(adminUser, { emailVerified: (event.currentTarget as HTMLInputElement).checked })} />
                      {adminUser.emailVerified ? tr('verifiedEmail') : tr('unverifiedEmail')}
                    </label>
                  </article>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      </section>

      <section class="panel">
        <div class="panel-title-row">
          <h3>{month} {tr('roster')}</h3>
        </div>
        <div class="summary-strip">
          <span>{tr('totalHours')}</span>
          <strong>{fmtHours(totalHours) || '0h'}</strong>
        </div>
        {#if schedules.length === 0}
          <p class="muted">{tr('noSchedule')}</p>
        {:else}
          <div class="schedule-list">
            {#each schedules as item}
              <article>
                <time>{item.shiftDate}</time>
                <div>
                  <strong>{shiftLabel(item)}</strong>
                  <span>{fmtTime(item.startTime)} {item.endTime ? `- ${fmtTime(item.endTime)}` : ''}</span>
                  {#if item.segments?.length}
                    <div class="segment-list">
                      {#each item.segments as segment}
                        <span><b>{segmentLabel(segment)}</b>{segmentMeta(segment)}</span>
                      {/each}
                    </div>
                  {/if}
                </div>
                <small>{scheduleMeta(item)}</small>
              </article>
            {/each}
          </div>
        {/if}
      </section>
    </section>
  </main>
{/if}
