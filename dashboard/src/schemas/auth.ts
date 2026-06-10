import { z } from 'zod'

export const LoginResponseSchema = z.object({
  token: z.string(),
  role: z.string(),
  must_change_password: z.boolean()
})

export const RefreshTokenResponseSchema = z.object({
  token: z.string(),
  role: z.string(),
  must_change_password: z.boolean()
})

const ErrorResponseSchema = z.object({
  error: z.string()
})
