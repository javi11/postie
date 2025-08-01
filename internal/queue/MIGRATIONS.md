# Database Migrations with Goose

This project uses [goose](https://github.com/pressly/goose) for database schema migrations.

## Overview

- Migrations are stored in `internal/queue/migrations/`
- Migrations are embedded in the binary using Go embed
- Automatic migration on application startup
- Legacy database detection and recreation

## Migration Files

Migration files follow the naming convention: `XXX_description.sql` where XXX is a sequential number.

Example: `001_initial_schema.sql`, `002_add_retry_limits.sql`

## File Structure

```
internal/queue/
├── migrations/
│   ├── 001_initial_schema.sql        # Initial database schema
│   └── 002_example_retry_limits.sql  # Example of adding new features
├── goose_migrations.go               # Goose integration
└── queue.go                         # Main queue logic
```

## Adding New Migrations

### 1. Create Migration File

Create a new SQL file in `internal/queue/migrations/`:

```sql
-- +goose Up
-- SQL for applying the migration
CREATE TABLE new_table (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

-- +goose Down  
-- SQL for rolling back the migration
DROP TABLE IF EXISTS new_table;
```

### 2. Migration Runs Automatically

The migration will run automatically when the application starts. No manual intervention needed.

### 3. Manual Migration Control (Optional)

You can also control migrations manually:

```go
// Get migration status
status, err := queue.GetMigrationStatus()
fmt.Printf("Current version: %d\n", status.CurrentVersion)

// Run pending migrations
err = queue.RunMigrations()

// Rollback last migration
err = queue.RollbackMigration()

// Migrate to specific version
err = queue.MigrateTo(5)

// Reset database (drop all + recreate)
err = queue.ResetDatabase()
```

## Legacy Database Handling

The system automatically detects legacy databases (created before goose) and recreates them:

1. **Detection**: Checks for `goqite` table without `goose_db_version` table
2. **Recreation**: Drops all existing tables and recreates using migrations
3. **Automatic**: Happens transparently on startup

## Best Practices

### 1. **Sequential Numbering**
Use sequential numbers: `001_`, `002_`, `003_`, etc.

### 2. **Descriptive Names**
Use clear, descriptive names:
- ✅ `003_add_user_preferences.sql`
- ❌ `003_changes.sql`

### 3. **Always Include Down Migration**
Always provide rollback SQL in the `+goose Down` section.

### 4. **Test Migrations**
Test both up and down migrations:
```bash
# Apply migration
go run main.go  # Migrations run automatically

# Test rollback (if needed)
# Use queue.RollbackMigration() in code
```

### 5. **SQLite Limitations**
SQLite doesn't support `DROP COLUMN`, so:
- **Adding columns**: ✅ Easy with `ALTER TABLE ADD COLUMN`
- **Removing columns**: ❌ Requires table recreation
- **Changing columns**: ❌ Requires table recreation

For complex schema changes, recreate the table:

```sql
-- +goose Up
-- Create new table with desired schema
CREATE TABLE goqite_new (
    id TEXT PRIMARY KEY,
    -- new schema here
);

-- Copy data
INSERT INTO goqite_new SELECT * FROM goqite;

-- Replace old table
DROP TABLE goqite;
ALTER TABLE goqite_new RENAME TO goqite;

-- +goose Down
-- Reverse the process
```

## Migration Examples

### Adding a Column
```sql
-- +goose Up
ALTER TABLE goqite ADD COLUMN new_field TEXT NOT NULL DEFAULT '';

-- +goose Down  
-- SQLite doesn't support DROP COLUMN
-- In production, you might need to recreate the table
```

### Creating a New Table
```sql
-- +goose Up
CREATE TABLE user_preferences (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    preference_key TEXT NOT NULL,
    preference_value TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ'))
);

CREATE INDEX user_preferences_user_idx ON user_preferences (user_id);

-- +goose Down
DROP INDEX IF EXISTS user_preferences_user_idx;
DROP TABLE IF EXISTS user_preferences;
```

### Adding Indexes
```sql
-- +goose Up
CREATE INDEX IF NOT EXISTS goqite_status_idx ON goqite (queue, received, priority);

-- +goose Down
DROP INDEX IF EXISTS goqite_status_idx;
```

## Troubleshooting

### Migration Fails
1. Check the SQL syntax
2. Verify the migration file format (`+goose Up` and `+goose Down`)
3. Check logs for specific error messages

### Legacy Database Issues
If legacy detection fails:
```go
// Force recreation
err := queue.RecreateDatabase()
```

### Reset Everything
To start fresh:
```go
// This will drop all tables and recreate from migrations
err := queue.ResetDatabase()
```

## Command Line Tools (Optional)

You can also use the goose CLI for advanced operations:

```bash
# Install goose CLI
go install github.com/pressly/goose/v3/cmd/goose@latest

# Create new migration
goose -dir internal/queue/migrations create add_new_feature sql

# Check status
goose -dir internal/queue/migrations sqlite3 ./database.db status

# Apply migrations
goose -dir internal/queue/migrations sqlite3 ./database.db up

# Rollback
goose -dir internal/queue/migrations sqlite3 ./database.db down
```

## Integration Notes

- Migrations are **embedded** in the binary (no external files needed)
- **Automatic execution** on application startup
- **Legacy database support** with automatic recreation
- **Thread-safe** migration execution
- **Transaction-based** migrations for safety