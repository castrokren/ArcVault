/**
 * API Contract Drift Check Script
 *
 * Validates that the running coordinator's API responses match the Zod schemas.
 * Run this before each release to catch contract drift early.
 *
 * Usage:
 *   npx tsx dashboard/scripts/check-contract.ts
 */

import * as api from '../src/api'
import { AgentListSchema } from '../src/schemas/agents'
import { JobListSchema } from '../src/schemas/jobs'
import { GroupListSchema } from '../src/schemas/groups'
import { VersionResponseSchema } from '../src/schemas/status'

const BASE_URL = process.env.API_URL || 'http://localhost:8080'

interface CheckResult {
  endpoint: string
  status: 'PASS' | 'FAIL'
  error?: string
}

const results: CheckResult[] = []

async function checkEndpoint(
  endpoint: string,
  schema: any,
  fetcher: () => Promise<any>
): Promise<void> {
  try {
    const data = await fetcher()
    const result = schema.safeParse(data)

    if (result.success) {
      results.push({ endpoint, status: 'PASS' })
      console.log(`✓ ${endpoint}`)
    } else {
      results.push({
        endpoint,
        status: 'FAIL',
        error: JSON.stringify(result.error.format(), null, 2)
      })
      console.log(`✗ ${endpoint}`)
      console.log(`  ${result.error.message}`)
    }
  } catch (err) {
    results.push({
      endpoint,
      status: 'FAIL',
      error: err instanceof Error ? err.message : String(err)
    })
    console.log(`✗ ${endpoint} (fetch failed)`)
    console.log(`  ${err instanceof Error ? err.message : String(err)}`)
  }
}

async function main() {
  console.log(`\n📋 API Contract Drift Check\n`)
  console.log(`Validating against: ${BASE_URL}\n`)

  // Set API base for fetches
  Object.defineProperty(window || global, 'fetch', {
    value: global.fetch,
    writable: true
  })

  // Check each audited endpoint
  await checkEndpoint(
    'GET /api/agents',
    AgentListSchema,
    () => fetch(`${BASE_URL}/api/agents`).then(r => r.json())
  )

  await checkEndpoint(
    'GET /api/jobs',
    JobListSchema,
    () => fetch(`${BASE_URL}/api/jobs`).then(r => r.json())
  )

  await checkEndpoint(
    'GET /api/groups',
    GroupListSchema,
    () => fetch(`${BASE_URL}/api/groups`).then(r => r.json())
  )

  await checkEndpoint(
    'GET /api/version',
    VersionResponseSchema,
    () => fetch(`${BASE_URL}/api/version`).then(r => r.json())
  )

  // Summary
  const passed = results.filter(r => r.status === 'PASS').length
  const failed = results.filter(r => r.status === 'FAIL').length

  console.log(`\n${'─'.repeat(50)}`)
  console.log(`Results: ${passed} passed, ${failed} failed\n`)

  if (failed > 0) {
    console.log('❌ Contract validation FAILED')
    process.exit(1)
  } else {
    console.log('✅ All contracts valid')
    process.exit(0)
  }
}

main()
