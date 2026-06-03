import { z } from 'zod'

export const GroupSchema = z.object({
  id: z.number(),
  name: z.string(),
  description: z.string(),
  agent_count: z.number()
})

export const GroupListSchema = z.array(GroupSchema)
