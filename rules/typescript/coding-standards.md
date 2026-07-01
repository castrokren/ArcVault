---
paths:
  - "**/*.ts"
  - "**/*.tsx"
  - "**/*.vue"
  - "**/*.js"
  - "**/*.jsx"
---
# TypeScript Coding Standards

## Naming Conventions

| Category | Convention | Example |
|---|---|---|
| Variables / functions | `camelCase` | `getUser()`, `isLoading` |
| Classes / components / types | `PascalCase` | `UserProfile`, `UserCard` |
| Constants (module-level) | `UPPER_SNAKE_CASE` | `MAX_RETRY_COUNT` |
| Files | `kebab-case` | `user-profile.ts` |

## TypeScript Config

- `strict: true` in `tsconfig.json` — enables all strict type-checking options
- `noImplicitAny: true` — never allow implicit `any`
- `strictNullChecks: true` — catch null/undefined access at compile time

## Type Annotations

- Prefer explicit return types on exported functions (API contract)
- Inline inference is fine for internal/local variables
- **Never use `any`** — use `unknown` and narrow with type guards

```typescript
// ❌ Bad
export function parseData(input: any): any { ... }

// ✅ Good
export function parseData(input: unknown): Record<string, unknown> { ... }
```

## Interfaces vs Types

- Use `interface` for object shapes (extensible, mergeable)
- Use `type` for unions, intersections, and primitives

```typescript
// ✅ interface for objects
interface User { id: number; name: string }

// ✅ type for unions
type Status = 'idle' | 'loading' | 'error'
```

## Exports

- Barrel exports (`index.ts`) at the directory level for clean imports
- Named exports over default exports (better refactoring, tree-shaking)
