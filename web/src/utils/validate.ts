export const usernameOk = (v: string) => /^[A-Za-z0-9_.-]{3,32}$/.test(v.trim())

export const passwordOk = (v: string) => v.length >= 8 && v.length <= 72

/** Empty is allowed: email is optional. */
export const emailRule = (v: string) => !v.trim() || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v.trim())
