use rusqlite::{Connection, Result as SqliteResult, params};
use std::path::Path;

pub struct LocalDb {
    conn: Connection,
}

impl LocalDb {
    pub fn new<P: AsRef<Path>>(path: P) -> SqliteResult<Self> {
        let conn = Connection::open(path)?;
        conn.execute_batch(
            "PRAGMA journal_mode = WAL;
             PRAGMA synchronous = NORMAL;
             PRAGMA cache_size = -64000;",
        )?;
        
        let db = LocalDb { conn };
        db.init_schema()?;
        Ok(db)
    }

    fn init_schema(&self) -> SqliteResult<()> {
        self.conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS functions (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                version INTEGER NOT NULL,
                entrypoint TEXT NOT NULL,
                memory_pages INTEGER NOT NULL,
                max_execution_ms INTEGER NOT NULL,
                artifact_url TEXT NOT NULL,
                sha256 TEXT NOT NULL,
                created_at INTEGER NOT NULL,
                UNIQUE(name, version)
            );

            CREATE TABLE IF NOT EXISTS deployments (
                id TEXT PRIMARY KEY,
                function_id TEXT NOT NULL,
                status TEXT NOT NULL,
                deployed_at INTEGER NOT NULL,
                FOREIGN KEY(function_id) REFERENCES functions(id)
            );

            CREATE TABLE IF NOT EXISTS cache_entries (
                id TEXT PRIMARY KEY,
                function_id TEXT NOT NULL,
                artifact_path TEXT NOT NULL,
                size_bytes INTEGER NOT NULL,
                sha256 TEXT NOT NULL,
                last_accessed INTEGER NOT NULL,
                created_at INTEGER NOT NULL,
                FOREIGN KEY(function_id) REFERENCES functions(id),
                UNIQUE(function_id)
            );

            CREATE INDEX IF NOT EXISTS idx_functions_name ON functions(name);
            CREATE INDEX IF NOT EXISTS idx_deployments_function_id ON deployments(function_id);
            CREATE INDEX IF NOT EXISTS idx_cache_entries_last_accessed ON cache_entries(last_accessed);",
        )?;
        Ok(())
    }

    pub fn get_connection(&self) -> &Connection {
        &self.conn
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_db_initialization() {
        let db = LocalDb::new(":memory:").unwrap();
        let conn = db.get_connection();
        
        let mut stmt = conn.prepare("SELECT name FROM sqlite_master WHERE type='table'").unwrap();
        let tables: Vec<String> = stmt.query_map([], |row| row.get(0))
            .unwrap()
            .filter_map(|r| r.ok())
            .collect();
        
        assert!(tables.contains(&"functions".to_string()));
        assert!(tables.contains(&"deployments".to_string()));
        assert!(tables.contains(&"cache_entries".to_string()));
    }
}
