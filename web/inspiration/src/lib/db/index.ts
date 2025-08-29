// Purpose: Database client configuration
// Context: Used to establish and manage database connections

import { createClient } from '@libsql/client';
import { drizzle } from 'drizzle-orm/libsql';
import * as schema from './schema';

// Create a SQLite database client
const client = createClient({
  url: 'file:static/data/db.sqlite',
});

// Create a Drizzle ORM instance
export const db = drizzle(client, { schema }); 