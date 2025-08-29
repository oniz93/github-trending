// Purpose: Optimize SQLite database for HTTP serving
// Context: Used in GitHub Actions to prepare the database for GitHub Pages

import { Database } from 'bun:sqlite';

const DB_PATH = 'static/data/db.sqlite';

async function optimizeDatabase() {
    console.log('Optimizing database for HTTP serving...');
    
    const db = new Database(DB_PATH);
    
    // Set optimal settings for HTTP serving
    db.run('PRAGMA journal_mode = DELETE');
    db.run('PRAGMA page_size = 1024');
    db.run('PRAGMA cache_size = 5000');
    db.run('PRAGMA temp_store = MEMORY');
    db.run('PRAGMA locking_mode = EXCLUSIVE');
    db.run('PRAGMA synchronous = NORMAL');
    
    // Optimize any FTS tables if they exist
    const ftsTablesQuery = `
        SELECT name FROM sqlite_master 
        WHERE type='table' 
        AND sql LIKE '%USING fts%'
    `;
    
    const ftsTables = db.query(ftsTablesQuery).all() as { name: string }[];
    for (const { name } of ftsTables) {
        console.log(`Optimizing FTS table: ${name}`);
        db.run(`INSERT INTO ${name}(${name}) VALUES('optimize')`);
    }
    
    // Vacuum the database to apply changes and compact
    console.log('Vacuuming database...');
    db.run('VACUUM');
    
    // Close the connection
    db.close();
    
    console.log('Database optimization complete!');
}

// Run the optimization
optimizeDatabase().catch(error => {
    console.error('Error optimizing database:', error);
    process.exit(1);
}); 