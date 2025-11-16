# Archive Notice

**Date**: October 17, 2025
**Reason**: Backend migrated from Node.js/TypeScript to Golang

## What's Archived

This directory contains documentation that was created during the initial backend planning phase when the project was planned to use Node.js/TypeScript/Express stack.

## What Changed

**Old Tech Stack (Planned)**:
- Node.js 20 LTS
- Express.js web framework
- TypeScript
- Prisma ORM
- Jest/Supertest for testing

**New Tech Stack (Implemented)**:
- Go 1.21+
- Gin web framework
- GORM ORM
- PostgreSQL
- Native Go testing

## Why the Change

The backend was migrated to Golang for the following reasons:
1. **Performance**: Go offers superior performance and lower memory footprint
2. **Concurrency**: Built-in goroutines for handling concurrent requests
3. **Deployment**: Single binary deployment, no runtime dependencies
4. **Type Safety**: Compile-time type checking without transpilation overhead
5. **Production Ready**: Excellent tooling and ecosystem for backend services

## What's Preserved

These documents are preserved for:
- **Historical Reference**: Understanding the original planning and decision-making process
- **Conceptual Value**: Many architectural concepts still apply regardless of language
- **Design Patterns**: Service layer, repository pattern, and other design decisions remain valid

## Current Documentation

For current, Golang-specific backend documentation, see:
- `/backend/README.md` - Main backend documentation
- `/backend/ARCHITECTURE.md` - Golang architecture patterns (when created)
- `/backend/docs/` - Updated planning documents

## Files in Archive

- `README_OLD.md` - Original Node.js backend README

## Note

While the technology changed, the core architectural principles and API contracts remain largely the same. The migration was primarily a technology choice, not a redesign of the application architecture.

---

**Archived On**: October 17, 2025
