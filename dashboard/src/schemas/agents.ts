import { z } from 'zod'

export const ProgressDataSchema = z.object({
  files_processed: z.number().int(),
  bytes_transferred: z.number().int(),
  total_files: z.number().int(),
  total_bytes: z.number().int(),
})

export const AgentSchema = z.object({
  id: z.string(),
  hostname: z.string(),
  os: z.string(),
  arch: z.string(),
  version: z.string(),
  status: z.string(),
  last_seen: z.string().nullable().optional(),
  registered_at: z.string(),
  rollback_available: z.boolean(),
})

export const AgentListSchema = z.array(AgentSchema)

export type Agent = z.infer<typeof AgentSchema>
export type ProgressData = z.infer<typeof ProgressDataSchema>
