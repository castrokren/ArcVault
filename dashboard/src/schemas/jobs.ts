import { z } from 'zod'

const ProgressDataSchema = z.object({
  files_processed: z.number(),
  bytes_transferred: z.number(),
  total_files: z.number(),
  total_bytes: z.number()
})

const JobSchema = z.object({
  id: z.string(),
  agent_id: z.string(),
  name: z.string(),
  source_path: z.string(),
  dest_path: z.string(),
  schedule: z.string().optional(),
  sync_flags: z.record(z.unknown()).optional(),
  status: z.string(),
  progress: ProgressDataSchema.optional(),
  created_at: z.string()
})

export const JobListSchema = z.array(JobSchema)
