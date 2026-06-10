import { z } from 'zod'

export const VersionResponseSchema = z.object({
  version: z.string(),
  build_time: z.string(),
  os: z.string(),
  arch: z.string(),
  go_version: z.string(),
  uptime: z.string()
})
