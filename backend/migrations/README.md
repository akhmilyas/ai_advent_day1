# Database Migrations

This directory contains database migration files managed by [golang-migrate](https://github.com/golang-migrate/migrate).

## Overview

Migrations provide version-controlled schema changes with rollback capability. Each migration consists of two files:
- `XXXXXX_name.up.sql` - applies the migration
- `XXXXXX_name.down.sql` - reverts the migration

## Migration Files

| Version | Description | Files |
|---------|-------------|-------|
| 000001 | Initial schema (users, conversations, messages) | `000001_initial_schema.{up,down}.sql` |
| 000002 | Add response_format and response_schema columns | `000002_add_response_format.{up,down}.sql` |
| 000003 | Add model field to messages | `000003_add_model_field.{up,down}.sql` |
| 000004 | Add temperature field to messages | `000004_add_temperature_field.{up,down}.sql` |
| 000005 | Add usage tracking fields (tokens, cost, latency) | `000005_add_usage_tracking.{up,down}.sql` |
| 000006 | Add provider field to messages | `000006_add_provider_field.{up,down}.sql` |
| 000007 | Add conversation_summaries table | `000007_add_conversation_summaries.{up,down}.sql` |

## How Migrations Work

1. **Automatic Application**: Migrations run automatically on backend startup via `postgres.RunMigrations()`
2. **Version Tracking**: The `schema_migrations` table tracks the current version and dirty state
3. **Idempotency**: Running migrations multiple times is safe - already-applied migrations are skipped

## Creating New Migrations

1. **Create migration files** with the next sequential number:
   ```bash
   touch backend/migrations/000008_add_new_feature.up.sql
   touch backend/migrations/000008_add_new_feature.down.sql
   ```

2. **Write the up migration** (schema change):
   ```sql
   -- 000008_add_new_feature.up.sql
   ALTER TABLE users
   ADD COLUMN avatar_url VARCHAR(500);
   ```

3. **Write the down migration** (rollback):
   ```sql
   -- 000008_add_new_feature.down.sql
   ALTER TABLE users
   DROP COLUMN IF EXISTS avatar_url;
   ```

4. **Rebuild and restart**:
   ```bash
   docker compose down
   docker compose build backend
   docker compose up -d
   ```

## Manual Migration Management

If you need to manually control migrations (not recommended for normal use):

### Check Current Version
```bash
docker compose exec postgres psql -U postgres -d chatapp -c "SELECT * FROM schema_migrations;"
```

### Apply Migrations Manually
```bash
# Install migrate CLI
brew install golang-migrate

# Run migrations up
migrate -path backend/migrations -database "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable" up

# Run migrations down (rollback one version)
migrate -path backend/migrations -database "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable" down 1

# Force version (if dirty state occurs)
migrate -path backend/migrations -database "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable" force VERSION
```

## Troubleshooting

### Dirty Database State
If migrations fail mid-execution, the database may be marked as "dirty":
```
error: Dirty database version X. Fix and force version.
```

**Solution**:
1. Fix the SQL error in the migration file
2. Manually fix the database if needed
3. Force the version:
   ```bash
   migrate -path backend/migrations -database "..." force X
   ```
4. Restart the backend

### Tables Already Exist
If converting from manual schema creation to migrations on an existing database:
1. Drop the database volume: `docker compose down -v`
2. Rebuild and restart: `docker compose build backend && docker compose up -d`

Alternatively, manually track the current schema version:
```sql
CREATE TABLE schema_migrations (version bigint not null primary key, dirty boolean not null);
INSERT INTO schema_migrations (version, dirty) VALUES (7, false);
```

## Best Practices

1. **Never modify applied migrations** - create a new migration instead
2. **Test down migrations** - ensure rollbacks work correctly
3. **Make migrations atomic** - each migration should be a single logical change
4. **Add comments** - explain why the change is being made
5. **Review before deployment** - verify SQL syntax and logic

## Migration Naming Convention

Format: `XXXXXX_description.{up,down}.sql`
- `XXXXXX`: 6-digit sequential number (e.g., 000008)
- `description`: snake_case description of the change
- `up`: applies the migration
- `down`: reverts the migration

Examples:
- `000008_add_user_roles.up.sql`
- `000009_create_analytics_table.up.sql`
- `000010_add_conversation_indexes.up.sql`

## References

- [golang-migrate documentation](https://github.com/golang-migrate/migrate/tree/master/database/postgres)
- [Migration best practices](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md)
